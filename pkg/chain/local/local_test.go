package local

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/keep-network/keep-ecdsa/pkg/ecdsa"
	"github.com/keep-network/keep-ecdsa/pkg/utils/byteutils"

	"github.com/ethereum/go-ethereum/common"
	eth "github.com/keep-network/keep-ecdsa/pkg/chain"
)

func TestOnBondedECDSAKeepCreated(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelCtx()

	chain := initializeLocalChain(ctx)
	eventFired := make(chan *eth.BondedECDSAKeepCreatedEvent)
	keepAddress := common.Address([20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	expectedEvent := &eth.BondedECDSAKeepCreatedEvent{
		KeepAddress: keepAddress,
	}

	subscription := chain.OnBondedECDSAKeepCreated(
		func(event *eth.BondedECDSAKeepCreatedEvent) {
			eventFired <- event
		},
	)
	defer subscription.Unsubscribe()

	err := chain.createKeep(keepAddress)
	if err != nil {
		t.Fatal(err)
	}

	select {
	case event := <-eventFired:
		if !reflect.DeepEqual(event, expectedEvent) {
			t.Fatalf(
				"unexpected keep creation event\nexpected: [%v]\nactual:   [%v]",
				expectedEvent,
				event,
			)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestOnSignatureRequested(t *testing.T) {
	ctx, cancelCtx := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelCtx()

	chain := initializeLocalChain(ctx)
	eventFired := make(chan *eth.SignatureRequestedEvent)
	keepAddress := common.Address([20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1})
	digest := [32]byte{1}

	err := chain.createKeep(keepAddress)
	if err != nil {
		t.Fatal(err)
	}

	var keepPubkey [64]byte
	rand.Read(keepPubkey[:])

	err = chain.SubmitKeepPublicKey(keepAddress, keepPubkey)
	if err != nil {
		t.Fatal(err)
	}

	subscription, err := chain.OnSignatureRequested(
		keepAddress,
		func(event *eth.SignatureRequestedEvent) {
			eventFired <- event
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Unsubscribe()

	err = chain.RequestSignature(keepAddress, digest)
	if err != nil {
		t.Fatal(err)
	}

	expectedEvent := &eth.SignatureRequestedEvent{
		Digest: digest,
	}

	select {
	case event := <-eventFired:
		if !reflect.DeepEqual(event, expectedEvent) {
			t.Fatalf(
				"unexpected signature requested event\nexpected: [%v]\nactual:   [%v]",
				expectedEvent,
				event,
			)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestSubmitKeepPublicKey(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	chain := initializeLocalChain(ctx)
	keepAddress := common.HexToAddress("0x41048F9B90290A2e96D07f537F3A7E97620E9e47")
	keepPublicKey := [64]byte{11, 12, 13, 14, 15, 16}
	expectedDuplicationError := fmt.Errorf(
		"public key already submitted for keep [%s]",
		keepAddress.String(),
	)

	err := chain.createKeep(keepAddress)
	if err != nil {
		t.Fatal(err)
	}

	err = chain.SubmitKeepPublicKey(
		keepAddress,
		keepPublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	onChainPubKey, err := chain.GetPublicKey(keepAddress)
	if err != nil {
		t.Fatal(err)
	}
	if hex.EncodeToString(keepPublicKey[:]) != hex.EncodeToString(onChainPubKey) {
		t.Errorf(
			"unexpected result\nexpected: [%+v]\nactual:   [%+v]",
			hex.EncodeToString(keepPublicKey[:]),
			hex.EncodeToString(onChainPubKey),
		)
	}

	err = chain.SubmitKeepPublicKey(
		keepAddress,
		keepPublicKey,
	)
	if !reflect.DeepEqual(expectedDuplicationError, err) {
		t.Errorf(
			"unexpected error\nexpected: [%+v]\nactual:   [%+v]",
			expectedDuplicationError,
			err,
		)
	}
}

func TestSubmitSignature(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	chain := initializeLocalChain(ctx)

	keepAddress := common.HexToAddress("0x41048F9B90290A2e96D07f537F3A7E97620E9e47")
	keepPublicKey := [64]byte{11, 12, 13, 14, 15, 16}

	err := chain.createKeep(keepAddress)
	if err != nil {
		t.Fatal(err)
	}

	err = chain.SubmitKeepPublicKey(
		keepAddress,
		keepPublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	digest := [32]byte{17, 18}

	err = chain.RequestSignature(keepAddress, digest)
	if err != nil {
		t.Fatal(err)
	}

	signature := &ecdsa.Signature{
		R:          big.NewInt(10),
		S:          big.NewInt(11),
		RecoveryID: 1,
	}

	err = chain.SubmitSignature(keepAddress, signature)
	if err != nil {
		t.Fatal(err)
	}

	events, err := chain.PastSignatureSubmittedEvents(keepAddress.Hex(), 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(events) != 1 {
		t.Errorf("there should be one signature submitted event")
	}

	expectedRBytes, _ := byteutils.BytesTo32Byte(signature.R.Bytes())
	expectedSBytes, _ := byteutils.BytesTo32Byte(signature.S.Bytes())
	expectedEvent := &eth.SignatureSubmittedEvent{
		Digest:      digest,
		R:           expectedRBytes,
		S:           expectedSBytes,
		RecoveryID:  1,
		BlockNumber: 0,
	}

	lastEvent := events[len(events)-1]

	if !reflect.DeepEqual(expectedEvent, lastEvent) {
		t.Fatalf(
			"unexpected signature submitted event\nexpected: [%+v]\nactual:   [%+v]",
			expectedEvent,
			lastEvent,
		)
	}
}

func TestIsAwaitingSignature(t *testing.T) {
	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	chain := initializeLocalChain(ctx)

	keepAddress := common.HexToAddress("0x41048F9B90290A2e96D07f537F3A7E97620E9e47")
	keepPublicKey := [64]byte{11, 12, 13, 14, 15, 16}

	err := chain.createKeep(keepAddress)
	if err != nil {
		t.Fatal(err)
	}

	err = chain.SubmitKeepPublicKey(
		keepAddress,
		keepPublicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	digest := [32]byte{17, 18}

	err = chain.RequestSignature(keepAddress, digest)
	if err != nil {
		t.Fatal(err)
	}

	isAwaitingSignature, err := chain.IsAwaitingSignature(keepAddress, digest)
	if !isAwaitingSignature {
		t.Error("keep should be awaiting for a signature for requested digest")
	}

	anotherDigest := [32]byte{18, 17}
	isAwaitingSignature, err = chain.IsAwaitingSignature(keepAddress, anotherDigest)
	if !isAwaitingSignature {
		t.Error("keep should not be awaiting for a signature for a not requested digest")
	}

	signature := &ecdsa.Signature{
		R:          big.NewInt(10),
		S:          big.NewInt(11),
		RecoveryID: 1,
	}

	err = chain.SubmitSignature(keepAddress, signature)
	if err != nil {
		t.Fatal(err)
	}

	isAwaitingSignature, err = chain.IsAwaitingSignature(keepAddress, digest)
	if !isAwaitingSignature {
		t.Error("keep should be awaiting for already provided signature")
	}
}

func initializeLocalChain(ctx context.Context) *localChain {
	return Connect(ctx).(*localChain)
}
