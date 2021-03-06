/* Generated by ts-generator ver. 0.0.8 */
/* tslint:disable */

import { Contract, ContractTransaction, EventFilter, Signer } from 'ethers'
import { Listener, Provider } from 'ethers/providers'
import { Arrayish, BigNumber, BigNumberish, Interface } from 'ethers/utils'
import {
  TransactionOverrides,
  TypedEventDescription,
  TypedFunctionDescription,
} from '.'

interface ArbFactoryInterface extends Interface {
  functions: {
    challengeFactoryAddress: TypedFunctionDescription<{
      encode([]: []): string
    }>

    createRollup: TypedFunctionDescription<{
      encode([
        _vmState,
        _gracePeriodTicks,
        _arbGasSpeedLimitPerTick,
        _maxExecutionSteps,
        _stakeRequirement,
        _stakeToken,
        _owner,
        _extraConfig,
      ]: [
        Arrayish,
        BigNumberish,
        BigNumberish,
        BigNumberish,
        BigNumberish,
        string,
        string,
        Arrayish
      ]): string
    }>

    globalInboxAddress: TypedFunctionDescription<{ encode([]: []): string }>

    rollupTemplate: TypedFunctionDescription<{ encode([]: []): string }>
  }

  events: {
    RollupCreated: TypedEventDescription<{
      encodeTopics([rollupAddress]: [null]): string[]
    }>
  }
}

export class ArbFactory extends Contract {
  connect(signerOrProvider: Signer | Provider | string): ArbFactory
  attach(addressOrName: string): ArbFactory
  deployed(): Promise<ArbFactory>

  on(event: EventFilter | string, listener: Listener): ArbFactory
  once(event: EventFilter | string, listener: Listener): ArbFactory
  addListener(eventName: EventFilter | string, listener: Listener): ArbFactory
  removeAllListeners(eventName: EventFilter | string): ArbFactory
  removeListener(eventName: any, listener: Listener): ArbFactory

  interface: ArbFactoryInterface

  functions: {
    challengeFactoryAddress(overrides?: TransactionOverrides): Promise<string>

    'challengeFactoryAddress()'(
      overrides?: TransactionOverrides
    ): Promise<string>

    createRollup(
      _vmState: Arrayish,
      _gracePeriodTicks: BigNumberish,
      _arbGasSpeedLimitPerTick: BigNumberish,
      _maxExecutionSteps: BigNumberish,
      _stakeRequirement: BigNumberish,
      _stakeToken: string,
      _owner: string,
      _extraConfig: Arrayish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    'createRollup(bytes32,uint128,uint128,uint64,uint128,address,address,bytes)'(
      _vmState: Arrayish,
      _gracePeriodTicks: BigNumberish,
      _arbGasSpeedLimitPerTick: BigNumberish,
      _maxExecutionSteps: BigNumberish,
      _stakeRequirement: BigNumberish,
      _stakeToken: string,
      _owner: string,
      _extraConfig: Arrayish,
      overrides?: TransactionOverrides
    ): Promise<ContractTransaction>

    globalInboxAddress(overrides?: TransactionOverrides): Promise<string>

    'globalInboxAddress()'(overrides?: TransactionOverrides): Promise<string>

    rollupTemplate(overrides?: TransactionOverrides): Promise<string>

    'rollupTemplate()'(overrides?: TransactionOverrides): Promise<string>
  }

  challengeFactoryAddress(overrides?: TransactionOverrides): Promise<string>

  'challengeFactoryAddress()'(overrides?: TransactionOverrides): Promise<string>

  createRollup(
    _vmState: Arrayish,
    _gracePeriodTicks: BigNumberish,
    _arbGasSpeedLimitPerTick: BigNumberish,
    _maxExecutionSteps: BigNumberish,
    _stakeRequirement: BigNumberish,
    _stakeToken: string,
    _owner: string,
    _extraConfig: Arrayish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  'createRollup(bytes32,uint128,uint128,uint64,uint128,address,address,bytes)'(
    _vmState: Arrayish,
    _gracePeriodTicks: BigNumberish,
    _arbGasSpeedLimitPerTick: BigNumberish,
    _maxExecutionSteps: BigNumberish,
    _stakeRequirement: BigNumberish,
    _stakeToken: string,
    _owner: string,
    _extraConfig: Arrayish,
    overrides?: TransactionOverrides
  ): Promise<ContractTransaction>

  globalInboxAddress(overrides?: TransactionOverrides): Promise<string>

  'globalInboxAddress()'(overrides?: TransactionOverrides): Promise<string>

  rollupTemplate(overrides?: TransactionOverrides): Promise<string>

  'rollupTemplate()'(overrides?: TransactionOverrides): Promise<string>

  filters: {
    RollupCreated(rollupAddress: null): EventFilter
  }

  estimate: {
    challengeFactoryAddress(
      overrides?: TransactionOverrides
    ): Promise<BigNumber>

    'challengeFactoryAddress()'(
      overrides?: TransactionOverrides
    ): Promise<BigNumber>

    createRollup(
      _vmState: Arrayish,
      _gracePeriodTicks: BigNumberish,
      _arbGasSpeedLimitPerTick: BigNumberish,
      _maxExecutionSteps: BigNumberish,
      _stakeRequirement: BigNumberish,
      _stakeToken: string,
      _owner: string,
      _extraConfig: Arrayish,
      overrides?: TransactionOverrides
    ): Promise<BigNumber>

    'createRollup(bytes32,uint128,uint128,uint64,uint128,address,address,bytes)'(
      _vmState: Arrayish,
      _gracePeriodTicks: BigNumberish,
      _arbGasSpeedLimitPerTick: BigNumberish,
      _maxExecutionSteps: BigNumberish,
      _stakeRequirement: BigNumberish,
      _stakeToken: string,
      _owner: string,
      _extraConfig: Arrayish,
      overrides?: TransactionOverrides
    ): Promise<BigNumber>

    globalInboxAddress(overrides?: TransactionOverrides): Promise<BigNumber>

    'globalInboxAddress()'(overrides?: TransactionOverrides): Promise<BigNumber>

    rollupTemplate(overrides?: TransactionOverrides): Promise<BigNumber>

    'rollupTemplate()'(overrides?: TransactionOverrides): Promise<BigNumber>
  }
}
