/*
 * Copyright 2020, Offchain Labs, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package gobridge

import (
	"context"
	"errors"
	"github.com/offchainlabs/arbitrum/packages/arb-util/hashing"
	"github.com/offchainlabs/arbitrum/packages/arb-util/machine"
	"github.com/offchainlabs/arbitrum/packages/arb-validator-core/arbbridge"
	"github.com/offchainlabs/arbitrum/packages/arb-validator-core/valprotocol"
	"math/big"
	"sync"

	"github.com/offchainlabs/arbitrum/packages/arb-util/common"
	"github.com/offchainlabs/arbitrum/packages/arb-util/value"
)

const VALID_CHILD_TYPE = 3

type rollupData struct {
	initVMHash      common.Hash
	VMstate         machine.Status
	state           EthState
	chainParams     valprotocol.ChainParams
	owner           common.Address
	events          map[*common.BlockId][]arbbridge.Event
	creation        *common.BlockId
	stakers         map[common.Address]*staker
	leaves          map[common.Hash]bool
	lastConfirmed   common.Hash
	contractAddress common.Address
	nextConfirmed   common.Hash
}

type arbRollup struct {
	rollup *rollupData
	Client *GoArbAuthClient
	mux    sync.Mutex
}

func newRollupContract(contractAddress common.Address, client *GoArbAuthClient) (*arbRollup, error) {
	ru, ok := client.arbFactoryContract.rollups[contractAddress]
	if !ok {
		return nil, errors.New("Rollup contract not found")
	}

	newGlobalInbox(contractAddress, client)
	roll := &arbRollup{
		rollup: ru,
		Client: client,
	}
	return roll, nil
}

func newRollup(con *arbFactory,
	address common.Address,
	vmState common.Hash,
	params valprotocol.ChainParams,
	owner common.Address,
) {
	events := make(map[*common.BlockId][]arbbridge.Event)
	vmProto := hashing.SoliditySHA3(
		hashing.Bytes32(vmState),
		hashing.Bytes32(value.NewEmptyTuple().Hash()),
		hashing.Uint256(big.NewInt(0)),
	)
	innerHash := hashing.SoliditySHA3(
		hashing.Bytes32(vmProto),
		hashing.Uint256(big.NewInt(0)),
		hashing.Uint256(big.NewInt(0)),
		hashing.Uint256(big.NewInt(0)),
	)
	initialNode := hashing.SoliditySHA3(
		hashing.Uint256(big.NewInt(0)),
		hashing.Bytes32(innerHash),
	)
	con.client.arbFactoryContract.rollups[address] = &rollupData{
		initVMHash:      vmState,
		VMstate:         machine.Extensive,
		state:           Uninitialized,
		chainParams:     params,
		owner:           owner,
		events:          events,
		creation:        con.client.getCurrentBlock(),
		stakers:         make(map[common.Address]*staker),
		leaves:          make(map[common.Hash]bool),
		lastConfirmed:   initialNode,
		contractAddress: address,
	}
	con.rollups[address].leaves[initialNode] = true

}

func (vm *arbRollup) PlaceStake(ctx context.Context, stakeAmount *big.Int, proof1 []common.Hash, proof2 []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()
	location := calculatePath(vm.rollup.lastConfirmed, proof1)
	leaf := calculatePath(location, proof2)
	if !vm.rollup.leaves[leaf] {
		return errors.New("invalid path proof")
	}
	if err := createStake(vm, stakeAmount, location); err != nil {
		return err
	}

	event := arbbridge.StakeCreatedEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: vm.Client.getCurrentBlock(),
		},
		Staker:   vm.Client.fromAddr,
		NodeHash: location,
	}
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: event,
	})
	return nil
}

func createStake(vm *arbRollup, stakeAmount *big.Int, location common.Hash) error {
	if stakeAmount != vm.rollup.chainParams.StakeRequirement {
		return errors.New("invalid stake amount")
	}
	if _, ok := vm.rollup.stakers[vm.Client.fromAddr]; ok {
		return errors.New("staker already exists")
	}
	vm.rollup.stakers[vm.Client.fromAddr] = &staker{location, vm.Client.getCurrentBlock().Height, false, stakeAmount}

	return nil
}

func refundStaker(vm *arbRollup, staker common.Address) {
	//refundStaker(stakerAddress);
	delete(vm.rollup.stakers, staker)
	// TODO:
	//transfer stake requirement
	// ???
	vm.Client.arbFactoryContract.rollups[vm.Client.Address()].events[vm.Client.getCurrentBlock()] = append(vm.Client.arbFactoryContract.rollups[vm.Client.Address()].events[vm.Client.getCurrentBlock()], arbbridge.StakeRefundedEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: vm.Client.getCurrentBlock(),
		},
		Staker: staker,
	})

}

func (vm *arbRollup) RecoverStakeConfirmed(ctx context.Context, proof []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()
	//bytes32 stakerLocation = getStakerLocation(msg.sender);
	//require(RollupUtils.calculatePath(stakerLocation, proof) == latestConfirmed(), RECOV_PATH_PROOF);
	//refundStaker(stakerAddress);

	staker, ok := vm.rollup.stakers[vm.Client.fromAddr]
	if !ok {
		return errors.New("staker not found")
	}

	if calculatePath(staker.location, proof) != vm.rollup.lastConfirmed {
		return errors.New("invalid path proof")
	}

	// refundStaker
	refundStaker(vm, vm.Client.fromAddr)

	//emit RollupStakeRefunded(contractAddress(_stakerAddress));
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: arbbridge.StakeRefundedEvent{
			ChainInfo: arbbridge.ChainInfo{
				BlockId: vm.Client.getCurrentBlock(),
			},
			Staker: vm.Client.fromAddr,
		},
	})

	return nil
}

func (vm *arbRollup) RecoverStakeOld(ctx context.Context, staker common.Address, proof []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()
	if len(proof) <= 0 {
		return errors.New("proof must be non-zero length")
	}
	//_recoverStakeConfirmed(stakerAddress, proof);
	//bytes32 stakerLocation = getStakerLocation(msg.sender);
	st, ok := vm.rollup.stakers[staker]
	if !ok {
		return errors.New("staker not found")
	}
	if calculatePath(st.location, proof) != vm.rollup.lastConfirmed {
		return errors.New("invalid path proof")
	}
	refundStaker(vm, staker)

	return nil
}

func (vm *arbRollup) RecoverStakeMooted(ctx context.Context, nodeHash common.Hash, staker common.Address, latestConfirmedProof []common.Hash, stakerProof []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()

	if latestConfirmedProof[0] == stakerProof[0] ||
		calculatePath(nodeHash, latestConfirmedProof) == vm.rollup.lastConfirmed ||
		calculatePath(nodeHash, stakerProof) != vm.rollup.stakers[vm.Client.fromAddr].location {
		return errors.New("Invalid conflict proof")
	}
	refundStaker(vm, staker)

	return nil
}

func (vm *arbRollup) RecoverStakePassedDeadline(
	ctx context.Context,
	stakerAddress common.Address,
	deadlineTicks *big.Int,
	disputableNodeHashVal common.Hash,
	childType uint64,
	vmProtoStateHash common.Hash,
	proof []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()

	leaf := calculatePath(vm.rollup.stakers[stakerAddress].location, proof)
	if !vm.rollup.leaves[leaf] {
		return errors.New("invalid leaf")
	}

	if common.TicksFromBlockNum(vm.Client.getCurrentBlock().Height).Val.Cmp(deadlineTicks) < 0 {
		return errors.New("Node is not passed deadline")
	}
	refundStaker(vm, stakerAddress)

	return nil
}

func (vm *arbRollup) MoveStake(ctx context.Context, proof1 []common.Hash, proof2 []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()

	location := vm.rollup.stakers[vm.Client.fromAddr].location
	newLocation := calculatePath(location, proof1)
	leaf := calculatePath(newLocation, proof2)
	if !vm.rollup.leaves[leaf] {
		return errors.New("MoveStake - invalid leaf")
	}
	vm.rollup.stakers[vm.Client.fromAddr].location = newLocation
	//emit RollupStakeRefunded(contractAddress(_stakerAddress));
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: arbbridge.StakeRefundedEvent{
			ChainInfo: arbbridge.ChainInfo{
				BlockId: vm.Client.getCurrentBlock(),
			},
			Staker: vm.Client.fromAddr,
		},
	})

	return nil
}

func (vm *arbRollup) PruneLeaf(ctx context.Context, from common.Hash, leafProof []common.Hash, ancProof []common.Hash) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()
	leaf := calculatePath(from, leafProof)
	if !vm.rollup.leaves[leaf] {
		return errors.New("PruneLeaf - invalid leaf")
	}
	if leafProof[0] == ancProof[0] ||
		calculatePath(from, ancProof) != vm.rollup.lastConfirmed {
		return errors.New("prune conflict")
	}
	delete(vm.rollup.leaves, leaf)

	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: arbbridge.PrunedEvent{
			ChainInfo: arbbridge.ChainInfo{
				BlockId: vm.Client.getCurrentBlock(),
			},
			Leaf: leaf,
		},
	})

	return nil
}

func (vm *arbRollup) PruneLeaves(ctx context.Context, opps []valprotocol.PruneParams) error {
	for _, opp := range opps {
		err := vm.PruneLeaf(ctx, opp.AncestorHash, opp.LeafProof, opp.AncProof)
		if err != nil {
			return err
		}
	}

	return nil
}

func (vm *arbRollup) MakeAssertion(
	ctx context.Context,

	prevPrevLeafHash common.Hash,
	prevDataHash common.Hash,
	prevDeadline common.TimeTicks,
	prevChildType valprotocol.ChildType,

	beforeState *valprotocol.VMProtoData,
	assertionParams *valprotocol.AssertionParams,
	assertionClaim *valprotocol.AssertionClaim,
	stakerProof []common.Hash,
) error {
	vm.mux.Lock()
	defer vm.mux.Unlock()

	protoHashBefore := beforeState.Hash()
	prevLeaf, _ := valprotocol.NodeHash(prevPrevLeafHash,
		protoHashBefore,
		prevDeadline,
		prevDataHash,
		prevChildType,
	)
	if !vm.rollup.leaves[prevLeaf] {
		return errors.New("makeAssertion - invalid leaf")
	}
	if vm.rollup.VMstate == machine.ErrorStop || vm.rollup.VMstate == machine.Halt {
		return errors.New("Can only disputable assert if machine is not errored or halted")
	}
	if assertionParams.NumSteps > vm.rollup.chainParams.MaxExecutionSteps {
		return errors.New("makeAssertion - Tried to execute too many steps")
	}
	if assertionParams.TimeBounds.IsValidTime(vm.Client.getCurrentBlock().Height) != nil {
		return errors.New("makeAssertion - Precondition: not within time bounds")
	}
	if assertionParams.ImportedMessageCount.Cmp(big.NewInt(0)) != 0 && !assertionClaim.AssertionStub.DidInboxInsn {
		return errors.New("makeAssertion - Imported messages without reading them")
	}
	if (vm.Client.globalInboxContract.inbox[vm.rollup.contractAddress]) != nil {
		inbox := vm.Client.globalInboxContract.inbox[vm.rollup.contractAddress]
		if assertionParams.ImportedMessageCount.Cmp(inbox.count.Sub(inbox.count, beforeState.InboxCount)) > 0 {
			return errors.New("makeAssertion - Tried to import more messages than exist in pending inbox")
		}
	} else {
		if assertionParams.ImportedMessageCount.Cmp(big.NewInt(0)) != 0 {
			return errors.New("makeAssertion - Tried to import more messages than exist in pending inbox")
		}
	}

	currentTicks := common.TicksFromBlockNum(vm.Client.getCurrentBlock().Height)
	deadlineTicks := currentTicks.Add(vm.rollup.chainParams.GracePeriod)
	if deadlineTicks.Cmp(prevDeadline) < 0 {
		return errors.New("Node is not passed deadline")
	}
	checkTimeTicks := assertionClaim.AssertionStub.NumGas / vm.rollup.chainParams.ArbGasSpeedLimitPerTick
	deadlineTicks = deadlineTicks.Add(common.TicksFromSeconds(int64(checkTimeTicks)))

	protoStateHash := hashing.SoliditySHA3(
		hashing.Bytes32(assertionClaim.AssertionStub.AfterHash),
		hashing.Bytes32(assertionClaim.AfterInboxTop),
		hashing.Uint256(beforeState.InboxCount.Add(beforeState.InboxCount, assertionParams.ImportedMessageCount)),
	)
	protoData := hashing.SoliditySHA3(
		hashing.Bytes32(assertionClaim.AssertionStub.LastMessageHash),
		hashing.Bytes32(assertionClaim.AssertionStub.LastLogHash),
	)

	var pendingTopCount *big.Int
	var pendingTopHash common.Hash
	globalInboxPending, ok := vm.Client.globalInboxContract.inbox[vm.rollup.contractAddress]
	if !ok {
		pendingTopCount = big.NewInt(0)
		pendingTopHash = value.NewEmptyTuple().Hash()
	} else {
		pendingTopCount = globalInboxPending.count
		pendingTopHash = globalInboxPending.value
	}
	left := new(big.Int).Add(beforeState.InboxCount, assertionParams.ImportedMessageCount)
	left = left.Sub(pendingTopCount, left)
	invPendingChallengeDataHash := valprotocol.InboxTopChallengeDataHash(
		assertionClaim.AfterInboxTop,
		pendingTopHash,
		left,
	)
	ticks := vm.rollup.chainParams.GracePeriod.Add(common.TicksFromBlockNum(common.NewTimeBlocks(big.NewInt(1))))
	invPendingProtoData := hashing.SoliditySHA3(
		hashing.Bytes32(invPendingChallengeDataHash),
		hashing.TimeTicks(ticks),
	)
	invalidPending, _ := valprotocol.NodeHash(prevLeaf,
		protoHashBefore,
		deadlineTicks,
		invPendingProtoData,
		valprotocol.InvalidInboxTopChildType)

	invMsgsChallengeDataHash := valprotocol.MessageChallengeDataHash(
		beforeState.InboxTop,
		assertionClaim.AfterInboxTop,
		value.NewEmptyTuple().Hash(),
		assertionClaim.ImportedMessagesSlice,
		assertionParams.ImportedMessageCount,
	)
	invMsgsProtoData := hashing.SoliditySHA3(
		hashing.Bytes32(invMsgsChallengeDataHash),
		hashing.TimeTicks(vm.rollup.chainParams.GracePeriod.Add(common.TicksFromBlockNum(common.NewTimeBlocks(big.NewInt(1))))),
	)
	invalidMessages, _ := valprotocol.NodeHash(prevLeaf,
		protoHashBefore,
		deadlineTicks,
		invMsgsProtoData,
		valprotocol.InvalidMessagesChildType)

	invExecChallengeDataHash := valprotocol.ExecutionDataHash(
		assertionParams.NumSteps,
		valprotocol.ExecutionPreconditionHash(beforeState.MachineHash, assertionParams.TimeBounds, assertionClaim.ImportedMessagesSlice),
		assertionClaim.AssertionStub.Hash(),
	)
	invExecProtoData := hashing.SoliditySHA3(
		hashing.Bytes32(invExecChallengeDataHash),
		hashing.TimeTicks(vm.rollup.chainParams.GracePeriod.Add(common.TimeTicks{new(big.Int).SetUint64(assertionClaim.AssertionStub.NumGas / vm.rollup.chainParams.ArbGasSpeedLimitPerTick)})),
	)
	invalidExecution, _ := valprotocol.NodeHash(prevLeaf,
		protoHashBefore,
		deadlineTicks,
		invExecProtoData,
		valprotocol.InvalidExecutionChildType,
	)

	valid, _ := valprotocol.NodeHash(prevLeaf,
		protoStateHash,
		deadlineTicks,
		protoData,
		valprotocol.ValidChildType,
	)

	vm.rollup.leaves[invalidPending] = true
	vm.rollup.leaves[invalidMessages] = true
	vm.rollup.leaves[invalidExecution] = true
	vm.rollup.leaves[valid] = true
	delete(vm.rollup.leaves, prevLeaf)

	event := arbbridge.AssertedEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: vm.Client.getCurrentBlock(),
		},
		PrevLeafHash:  prevLeaf,
		Params:        assertionParams,
		Claim:         assertionClaim,
		MaxInboxTop:   beforeState.InboxTop,
		MaxInboxCount: beforeState.InboxCount,
	}
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: event,
	})

	if calculatePath(vm.rollup.stakers[vm.Client.fromAddr].location, stakerProof) != prevLeaf {
		return errors.New("invalid staker location proof")
	}
	vm.rollup.stakers[vm.Client.fromAddr].location = valid
	vm.rollup.nextConfirmed = valid
	stakeMovedEvent := arbbridge.StakeMovedEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: vm.Client.getCurrentBlock(),
		},
		Staker:   vm.Client.fromAddr,
		Location: valid,
	}
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: stakeMovedEvent,
	})

	return nil
}

func (vm *arbRollup) Confirm(ctx context.Context, opp *valprotocol.ConfirmOpportunity) error {
	nodeOpps := opp.Nodes
	nodeCount := len(nodeOpps)
	lastMsg := common.Hash{}
	initalProtoStateHash := nodeOpps[0].StateHash()
	var lastLogHash common.Hash
	var nodeDataHash common.Hash
	validNum := 0
	invalidNum := 0
	confNode := opp.CurrentLatestConfirmed
	for _, opp := range nodeOpps {
		var linkType valprotocol.ChildType
		switch opp := opp.(type) {
		case valprotocol.ConfirmValidOpportunity:
			{
				linkType = VALID_CHILD_TYPE
				//logsAcc = append(logsAcc, opp.LogsAcc)
				lastLogHash = opp.LogsAcc
				if len(opp.Messages) > 0 {
					lastMsg = opp.Messages[len(opp.Messages)-1].Hash()
				}
				nodeDataHash = hashing.SoliditySHA3(
					hashing.Bytes32(lastMsg),
					hashing.Bytes32(lastLogHash),
				)
				validNum++
			}
		case valprotocol.ConfirmInvalidOpportunity:
			linkType = valprotocol.MaxInvalidChildType
			nodeDataHash = opp.ChallengeNodeData
			invalidNum++
		}
		confNode, _ = valprotocol.NodeHash(
			confNode,
			initalProtoStateHash,
			opp.Deadline(),
			nodeDataHash,
			linkType,
		)
	}

	if common.TicksFromBlockNum(vm.Client.LastMinedBlock.Height).Cmp(opp.Nodes[nodeCount-1].Deadline()) == -1 {
		panic("Node is not passed deadline")
		return errors.New("Node is not passed deadline")
	}

	activeCount := 0
	for i, staker := range opp.StakerAddresses {
		if !calculatePath(confNode, opp.StakerProofs[i]).Equals(vm.rollup.stakers[staker].location) {
			return errors.New("at least one active staker disagrees")
		}
		activeCount++
	}

	if activeCount == 0 {
		return errors.New("There must be at least one staker")
	}
	vm.rollup.lastConfirmed = confNode

	confirmedEvent := arbbridge.ConfirmedEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: vm.Client.getCurrentBlock(),
		},
		NodeHash: confNode,
	}
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: confirmedEvent,
	})

	confirmedAssertionEvent := arbbridge.ConfirmedAssertionEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: vm.Client.getCurrentBlock(),
		},
	}
	vm.Client.pubMsg(nil, arbbridge.MaybeEvent{
		Event: confirmedAssertionEvent,
	})
	return nil
}

func (vm *arbRollup) StartChallenge(
	ctx context.Context,
	asserterAddress common.Address,
	challengerAddress common.Address,
	prevNode common.Hash,
	disputableDeadline *big.Int,
	asserterPosition valprotocol.ChildType,
	challengerPosition valprotocol.ChildType,
	asserterVMProtoHash common.Hash,
	challengerVMProtoHash common.Hash,
	asserterProof []common.Hash,
	challengerProof []common.Hash,
	asserterNodeHash common.Hash,
	challengerDataHash common.Hash,
	challengerPeriodTicks common.TimeTicks,
) error {
	eth := vm.Client
	asserter, ok := vm.rollup.stakers[asserterAddress]
	if !ok {
		return errors.New("unknown asserter")
	}
	challenger, ok := vm.rollup.stakers[challengerAddress]
	if !ok {
		return errors.New("unknown challenger")
	}

	if asserter.creationTimeBlocks.AsInt().Cmp(disputableDeadline) == 1 {
		return errors.New("asserter staked after deadline")
	}
	if challenger.creationTimeBlocks.AsInt().Cmp(disputableDeadline) == 1 {
		return errors.New("challenger staked after deadline")
	}
	if asserter.inChallenge {
		return errors.New("asserter already in challenge")
	}
	if challenger.inChallenge {
		return errors.New("challenger already in challenge")
	}
	if asserterPosition <= challengerPosition {
		return errors.New("Child types must be ordere")
	}

	assnodedata, _ := valprotocol.NodeHash(
		prevNode,
		asserterVMProtoHash,
		common.TimeTicks{disputableDeadline},
		asserterNodeHash,
		asserterPosition,
	)
	if calculatePath(assnodedata, asserterProof) != asserter.location {
		return errors.New("Challenge asserter proof error")
	}

	chalnodedata, _ := valprotocol.NodeHash(
		prevNode,
		challengerVMProtoHash,
		common.TimeTicks{disputableDeadline},
		challengerDataHash,
		challengerPosition,
	)
	if calculatePath(chalnodedata, challengerProof) != challenger.location {
		return errors.New("Challenge challenger proof error")
	}

	asserter.inChallenge = true
	challenger.inChallenge = true

	// generate contractAddress
	newAddr := eth.getNextAddress()
	eth.challenges[newAddr] = &challengeData{deadline: common.TimeTicks{disputableDeadline}, challengerDataHash: challengerDataHash}
	// initialize bisection
	//save data
	// deadline = current + challenge period
	eth.challenges[newAddr].deadline = common.TicksFromBlockNum(eth.LastMinedBlock.Height).Add(challengerPeriodTicks)
	// emit InitiatedChallenge
	InitiateChallengeEvent := arbbridge.InitiateChallengeEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: eth.getCurrentBlock(),
		},
		Deadline: eth.challenges[newAddr].deadline,
	}
	eth.pubMsg(nil, arbbridge.MaybeEvent{
		Event: InitiateChallengeEvent,
	})

	ChallengeStartedEvent := arbbridge.ChallengeStartedEvent{
		ChainInfo: arbbridge.ChainInfo{
			BlockId: eth.getCurrentBlock(),
		},
		Asserter:          asserterAddress,
		Challenger:        challengerAddress,
		ChallengeType:     asserterPosition,
		ChallengeContract: newAddr,
	}
	eth.pubMsg(nil, arbbridge.MaybeEvent{
		Event: ChallengeStartedEvent,
	})
	return nil
}

func (vm *arbRollup) IsStaked(address common.Address) (bool, error) {
	return false, nil
}
