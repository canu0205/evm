package integration

import (
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/tests/integration/x/vm"
	testconstants "github.com/cosmos/evm/testutil/constants"
	utiltx "github.com/cosmos/evm/testutil/tx"
	"github.com/cosmos/evm/x/vm/keeper/testdata"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
)

var templateAccessListTx = &ethtypes.AccessListTx{
	GasPrice: big.NewInt(1),
	Gas:      21000,
	To:       &common.Address{},
	Value:    big.NewInt(0),
	Data:     []byte{},
}

var templateLegacyTx = &ethtypes.LegacyTx{
	GasPrice: big.NewInt(1),
	Gas:      21000,
	To:       &common.Address{},
	Value:    big.NewInt(0),
	Data:     []byte{},
}

var templateDynamicFeeTx = &ethtypes.DynamicFeeTx{
	GasFeeCap: big.NewInt(10),
	GasTipCap: big.NewInt(2),
	Gas:       21000,
	To:        &common.Address{},
	Value:     big.NewInt(0),
	Data:      []byte{},
}

func newSignedEthTx(
	txData ethtypes.TxData,
	nonce uint64,
	addr sdk.Address,
	krSigner keyring.Signer,
	ethSigner ethtypes.Signer,
) (*ethtypes.Transaction, error) {
	var ethTx *ethtypes.Transaction
	switch txData := txData.(type) {
	case *ethtypes.AccessListTx:
		txData.Nonce = nonce
		ethTx = ethtypes.NewTx(txData)
	case *ethtypes.LegacyTx:
		txData.Nonce = nonce
		ethTx = ethtypes.NewTx(txData)
	case *ethtypes.DynamicFeeTx:
		txData.Nonce = nonce
		ethTx = ethtypes.NewTx(txData)
	default:
		return nil, errors.New("unknown transaction type")
	}

	sig, _, err := krSigner.SignByAddress(addr, ethTx.Hash().Bytes(), signingtypes.SignMode_SIGN_MODE_TEXTUAL)
	if err != nil {
		return nil, err
	}

	ethTx, err = ethTx.WithSignature(ethSigner, sig)
	if err != nil {
		return nil, err
	}

	return ethTx, nil
}

func newEthMsgTx(
	nonce uint64,
	address common.Address,
	krSigner keyring.Signer,
	ethSigner ethtypes.Signer,
	txType byte,
	data []byte,
	accessList ethtypes.AccessList,
) (*evmtypes.MsgEthereumTx, *big.Int, error) {
	var (
		ethTx   *ethtypes.Transaction
		baseFee *big.Int
	)
	switch txType {
	case ethtypes.LegacyTxType:
		templateLegacyTx.Nonce = nonce
		if data != nil {
			templateLegacyTx.Data = data
		}
		ethTx = ethtypes.NewTx(templateLegacyTx)
	case ethtypes.AccessListTxType:
		templateAccessListTx.Nonce = nonce
		if data != nil {
			templateAccessListTx.Data = data
		} else {
			templateAccessListTx.Data = []byte{}
		}

		templateAccessListTx.AccessList = accessList
		ethTx = ethtypes.NewTx(templateAccessListTx)
	case ethtypes.DynamicFeeTxType:
		templateDynamicFeeTx.Nonce = nonce

		if data != nil {
			templateAccessListTx.Data = data
		} else {
			templateAccessListTx.Data = []byte{}
		}
		templateAccessListTx.AccessList = accessList
		ethTx = ethtypes.NewTx(templateDynamicFeeTx)
		baseFee = big.NewInt(3)
	default:
		return nil, baseFee, errors.New("unsupported tx type")
	}

	msg := &evmtypes.MsgEthereumTx{}
	err := msg.FromEthereumTx(ethTx)
	if err != nil {
		return nil, nil, err
	}

	msg.From = address.Hex()

	return msg, baseFee, msg.Sign(ethSigner, krSigner)
}

func newNativeMessage(
	nonce uint64,
	blockHeight int64,
	address common.Address,
	cfg *params.ChainConfig,
	krSigner keyring.Signer,
	ethSigner ethtypes.Signer,
	txType byte,
	data []byte,
	accessList ethtypes.AccessList,
) (*core.Message, error) {
	msgSigner := ethtypes.MakeSigner(cfg, big.NewInt(blockHeight), 10000000)

	msg, baseFee, err := newEthMsgTx(nonce, address, krSigner, ethSigner, txType, data, accessList)
	if err != nil {
		return nil, err
	}

	m, err := msg.AsMessage(msgSigner, baseFee)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// DoBenchmarkWithCreateEvmd wraps vm.DoBenchmark pattern with CreateEvmd
type TxBuilder func(suite *vm.KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx

func doBenchmarkWithCreateEvmd(b *testing.B, txBuilder TxBuilder) {
	b.Helper()
	suite := &vm.KeeperTestSuite{
		Create: CreateEvmd,
	}
	suite.SetupTest()

	amt := sdk.Coins{sdk.NewInt64Coin(testconstants.ExampleAttoDenom, 1000000000000000000)}
	err := suite.Network.App.GetBankKeeper().MintCoins(suite.Network.GetContext(), evmtypes.ModuleName, amt)
	require.NoError(b, err)
	err = suite.Network.App.GetBankKeeper().SendCoinsFromModuleToAccount(suite.Network.GetContext(), evmtypes.ModuleName, suite.Keyring.GetAddr(0).Bytes(), amt)
	require.NoError(b, err)

	contractAddr := suite.DeployTestContract(b, suite.Network.GetContext(), suite.Keyring.GetAddr(0), sdkmath.NewIntWithDecimal(1000, 18).BigInt())
	err = suite.Network.NextBlock()
	require.NoError(b, err)

	krSigner := utiltx.NewSigner(suite.Keyring.GetPrivKey(0))
	msg := txBuilder(suite, contractAddr)
	msg.From = suite.Keyring.GetAddr(0).Hex()
	err = msg.Sign(ethtypes.LatestSignerForChainID(evmtypes.GetEthChainConfig().ChainID), krSigner)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.Network.GetContext().CacheContext()

		// deduct fee first
		txData, err := evmtypes.UnpackTxData(msg.Data)
		require.NoError(b, err)

		fees := sdk.Coins{sdk.NewCoin(suite.EvmDenom(), sdkmath.NewIntFromBigInt(txData.Fee()))}
		err = authante.DeductFees(suite.Network.App.GetBankKeeper(), suite.Network.GetContext(), suite.Network.App.GetAccountKeeper().GetAccount(ctx, msg.GetFrom()), fees)
		require.NoError(b, err)

		rsp, err := suite.Network.App.GetEVMKeeper().EthereumTx(ctx, msg)
		require.NoError(b, err)
		require.False(b, rsp.Failed())
	}
}

// State transition benchmarks
func BenchmarkApplyTransaction(b *testing.B) {
	suite := vm.KeeperTestSuite{
		Create:         CreateEvmd,
		EnableLondonHF: true,
	}
	suite.SetupTest()

	ethSigner := ethtypes.LatestSignerForChainID(evmtypes.GetEthChainConfig().ChainID)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		addr := suite.Keyring.GetAddr(0)
		krSigner := utiltx.NewSigner(suite.Keyring.GetPrivKey(0))
		tx, err := newSignedEthTx(templateAccessListTx,
			suite.Network.App.GetEVMKeeper().GetNonce(suite.Network.GetContext(), addr),
			sdk.AccAddress(addr.Bytes()),
			krSigner,
			ethSigner,
		)
		require.NoError(b, err)

		b.StartTimer()
		resp, err := suite.Network.App.GetEVMKeeper().ApplyTransaction(suite.Network.GetContext(), tx)
		b.StopTimer()

		require.NoError(b, err)
		require.False(b, resp.Failed())
	}
}

func BenchmarkApplyMessage(b *testing.B) {
	suite := vm.KeeperTestSuite{
		Create:         CreateEvmd,
		EnableLondonHF: true,
	}
	suite.SetupTest()

	ethCfg := evmtypes.GetEthChainConfig()
	signer := ethtypes.LatestSignerForChainID(ethCfg.ChainID)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		addr := suite.Keyring.GetAddr(0)
		krSigner := utiltx.NewSigner(suite.Keyring.GetPrivKey(0))
		m, err := newNativeMessage(
			suite.Network.App.GetEVMKeeper().GetNonce(suite.Network.GetContext(), addr),
			suite.Network.GetContext().BlockHeight(),
			addr,
			ethCfg,
			krSigner,
			signer,
			ethtypes.AccessListTxType,
			nil,
			nil,
		)
		require.NoError(b, err)

		b.StartTimer()
		resp, err := suite.Network.App.GetEVMKeeper().ApplyMessage(suite.Network.GetContext(), *m, nil, true)
		b.StopTimer()

		require.NoError(b, err)
		require.False(b, resp.Failed())
	}
}

func BenchmarkSetParams(b *testing.B) {
	suite := vm.KeeperTestSuite{
		Create:         CreateEvmd,
		EnableLondonHF: true,
	}
	suite.SetupTest()

	defaultParams := evmtypes.DefaultParams()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = suite.Network.App.GetEVMKeeper().SetParams(suite.Network.GetContext(), defaultParams)
	}
}

func BenchmarkGetParams(b *testing.B) {
	suite := vm.KeeperTestSuite{
		Create:         CreateEvmd,
		EnableLondonHF: true,
	}
	suite.SetupTest()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = suite.Network.App.GetEVMKeeper().GetParams(suite.Network.GetContext())
	}
}

// Token operation benchmarks - using the simplified pattern
func BenchmarkTokenTransfer(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err, "failed to load erc20 contract")

	doBenchmarkWithCreateEvmd(b, func(suite *vm.KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("transfer", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.Network.App.GetEVMKeeper().GetNonce(suite.Network.GetContext(), suite.Keyring.GetAddr(0))
		ethTxParams := &evmtypes.EvmTxArgs{
			ChainID:  evmtypes.GetEthChainConfig().ChainID,
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}

func BenchmarkTokenMint(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err, "failed to load erc20 contract")

	doBenchmarkWithCreateEvmd(b, func(suite *vm.KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("mint", common.HexToAddress("0x378c50D9264C63F3F92B806d4ee56E9D86FfB3Ec"), big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.Network.App.GetEVMKeeper().GetNonce(suite.Network.GetContext(), suite.Keyring.GetAddr(0))
		ethTxParams := &evmtypes.EvmTxArgs{
			ChainID:  evmtypes.GetEthChainConfig().ChainID,
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 410000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}

func BenchmarkMessageCall(b *testing.B) {
	// Set up message call contract with CreateEvmd
	suite := &vm.KeeperTestSuite{
		Create: CreateEvmd,
	}
	suite.SetupTest()

	amt := sdk.Coins{sdk.NewInt64Coin(testconstants.ExampleAttoDenom, 1000000000000000000)}
	err := suite.Network.App.GetBankKeeper().MintCoins(suite.Network.GetContext(), evmtypes.ModuleName, amt)
	require.NoError(b, err)
	err = suite.Network.App.GetBankKeeper().SendCoinsFromModuleToAccount(suite.Network.GetContext(), evmtypes.ModuleName, suite.Keyring.GetAddr(0).Bytes(), amt)
	require.NoError(b, err)

	contract := suite.DeployTestMessageCall(b)
	err = suite.Network.NextBlock()
	require.NoError(b, err)

	messageCallContract, err := testdata.LoadMessageCallContract()
	require.NoError(b, err, "failed to load message call contract")

	input, err := messageCallContract.ABI.Pack("benchmarkMessageCall", big.NewInt(10000))
	require.NoError(b, err)
	nonce := suite.Network.App.GetEVMKeeper().GetNonce(suite.Network.GetContext(), suite.Keyring.GetAddr(0))
	ethCfg := evmtypes.GetEthChainConfig()
	ethTxParams := &evmtypes.EvmTxArgs{
		ChainID:  ethCfg.ChainID,
		Nonce:    nonce,
		To:       &contract,
		Amount:   big.NewInt(0),
		GasLimit: 25000000,
		GasPrice: big.NewInt(1),
		Input:    input,
	}
	msg := evmtypes.NewTx(ethTxParams)

	msg.From = suite.Keyring.GetAddr(0).Hex()
	krSigner := utiltx.NewSigner(suite.Keyring.GetPrivKey(0))
	err = msg.Sign(ethtypes.LatestSignerForChainID(ethCfg.ChainID), krSigner)
	require.NoError(b, err)

	b.ResetTimer()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ctx, _ := suite.Network.GetContext().CacheContext()

		// deduct fee first
		txData, err := evmtypes.UnpackTxData(msg.Data)
		require.NoError(b, err)

		fees := sdk.Coins{sdk.NewCoin(suite.EvmDenom(), sdkmath.NewIntFromBigInt(txData.Fee()))}
		err = authante.DeductFees(suite.Network.App.GetBankKeeper(), suite.Network.GetContext(), suite.Network.App.GetAccountKeeper().GetAccount(ctx, msg.GetFrom()), fees)
		require.NoError(b, err)

		rsp, err := suite.Network.App.GetEVMKeeper().EthereumTx(ctx, msg)
		require.NoError(b, err)
		require.False(b, rsp.Failed())
	}
}

func BenchmarkEmitLogs(b *testing.B) {
	erc20Contract, err := testdata.LoadERC20Contract()
	require.NoError(b, err, "failed to load erc20 contract")

	doBenchmarkWithCreateEvmd(b, func(suite *vm.KeeperTestSuite, contract common.Address) *evmtypes.MsgEthereumTx {
		input, err := erc20Contract.ABI.Pack("benchmarkLogs", big.NewInt(1000))
		require.NoError(b, err)
		nonce := suite.Network.App.GetEVMKeeper().GetNonce(suite.Network.GetContext(), suite.Keyring.GetAddr(0))
		ethTxParams := &evmtypes.EvmTxArgs{
			ChainID:  evmtypes.GetEthChainConfig().ChainID,
			Nonce:    nonce,
			To:       &contract,
			Amount:   big.NewInt(0),
			GasLimit: 4100000,
			GasPrice: big.NewInt(1),
			Input:    input,
		}
		return evmtypes.NewTx(ethTxParams)
	})
}
