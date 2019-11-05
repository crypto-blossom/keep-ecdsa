// Package tecdsa defines Keep tECDSA protocol.
package tecdsa

import (
	crand "crypto/rand"
	"fmt"

	"github.com/ipfs/go-log"

	"github.com/keep-network/keep-common/pkg/persistence"
	"github.com/keep-network/keep-tecdsa/pkg/chain/eth"
	"github.com/keep-network/keep-tecdsa/pkg/ecdsa"
	"github.com/keep-network/keep-tecdsa/pkg/registry"
)

var logger = log.Logger("keep-tecdsa")

// TECDSA holds an interface to interact with the blockchain.
type TECDSA struct {
	EthereumChain eth.Handle
}

// Initialize initializes the tECDSA client with rules related to events handling.
func Initialize(
	ethereumChain eth.Handle,
	persistence persistence.Handle,
) {
	keepsRegistry := registry.NewKeepsRegistry(persistence)

	client := &client{
		ethereumChain: ethereumChain,
		keepsRegistry: keepsRegistry,
	}

	// Load current keeps signers from storage and register for signing events.
	keepsRegistry.LoadExistingKeeps()

	for _, keepAddress := range keepsRegistry.GetKeepsAddresses() {
		client.registerForSignEvents(keepAddress)
	}

	// Watch for new keeps creation.
	ethereumChain.OnECDSAKeepCreated(func(event *eth.ECDSAKeepCreatedEvent) {
		logger.Infof(
			"new keep created with address: [%s]",
			event.KeepAddress.String(),
		)

		if event.IsMember(ethereumChain.Address()) {
			go func() {
				if err := client.generateSignerForKeep(event.KeepAddress); err != nil {
					logger.Errorf("signer generation failed: [%v]", err)
					return
				}

				client.registerForSignEvents(event.KeepAddress)
			}()
		}
	})

	// Register client as a candidate member for keep.
	err := ethereumChain.RegisterAsMemberCandidate()
	if err != nil {
		logger.Errorf("failed to register member: [%v]", err)
	} else {
		logger.Infof("client registered as member candidate in keep factory")
	}
}

// RegisterForSignEvents registers for signature requested events emitted by
// specific keep contract.
func (t *TECDSA) RegisterForSignEvents(
	keepAddress eth.KeepAddress,
	signer *ecdsa.Signer,
) {
	t.EthereumChain.OnSignatureRequested(
		keepAddress,
		func(signatureRequestedEvent *eth.SignatureRequestedEvent) {
			logger.Debugf(
				"new signature requested from keep [%s] for digest: [%+x]",
				keepAddress.String(),
				signatureRequestedEvent.Digest,
			)

			go func() {
				err := t.calculateSignatureForKeep(
					keepAddress,
					signer,
					signatureRequestedEvent.Digest,
				)

				if err != nil {
					logger.Errorf("signature calculation failed: [%v]", err)
				}
			}()
		},
	)
}

// GenerateSignerForKeep generates a new signer with ECDSA key pair. It publishes
// the signer's public key to the keep.
func (t *TECDSA) GenerateSignerForKeep(
	keepAddress eth.KeepAddress,
) (*ecdsa.Signer, error) {
	signer, err := generateSigner()

	logger.Debugf(
		"generated signer with public key: [%x]",
		signer.PublicKey().Marshal(),
	)

	// Publish signer's public key on ethereum blockchain in a specific keep
	// contract.
	serializedPublicKey, err := eth.SerializePublicKey(signer.PublicKey())
	if err != nil {
		return nil, fmt.Errorf("failed to serialize public key: [%v]", err)
	}

	err = t.EthereumChain.SubmitKeepPublicKey(
		keepAddress,
		serializedPublicKey,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to submit public key: [%v]", err)
	}

	logger.Debugf(
		"submitted public key to the keep [%s]: [%x]",
		keepAddress.String(),
		serializedPublicKey,
	)

	return signer, nil
}

func generateSigner() (*ecdsa.Signer, error) {
	privateKey, err := ecdsa.GenerateKey(crand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: [%v]", err)
	}

	return ecdsa.NewSigner(privateKey), nil
}

func (t *TECDSA) calculateSignatureForKeep(
	keepAddress eth.KeepAddress,
	signer *ecdsa.Signer,
	digest [32]byte,
) error {
	signature, err := signer.CalculateSignature(crand.Reader, digest[:])

	logger.Debugf(
		"signature calculated:\nr: [%#x]\ns: [%#x]\nrecovery ID: [%d]\n",
		signature.R,
		signature.S,
		signature.RecoveryID,
	)

	err = t.EthereumChain.SubmitSignature(keepAddress, digest, signature)
	if err != nil {
		return fmt.Errorf("failed to submit signature: [%v]", err)
	}

	logger.Infof("submitted signature for digest: [%+x]", digest)

	return nil
}
