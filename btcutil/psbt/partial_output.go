package psbt

import (
	"bytes"
	"io"
	"sort"

	"github.com/dogesuite/doged/wire"
)

// POutput is a struct encapsulating all the data that can be attached
// to any specific output of the PSBT.
type POutput struct {
	RedeemScript           []byte
	WitnessScript          []byte
	Bip32Derivation        []*Bip32Derivation
	TaprootInternalKey     []byte
	TaprootTapTree         []byte
	TaprootBip32Derivation []*TaprootBip32Derivation
}

// NewPsbtOutput creates an instance of PsbtOutput; the three parameters
// redeemScript, witnessScript and Bip32Derivation are all allowed to be
// `nil`.
func NewPsbtOutput(redeemScript []byte, witnessScript []byte,
	bip32Derivation []*Bip32Derivation) *POutput {
	return &POutput{
		RedeemScript:    redeemScript,
		WitnessScript:   witnessScript,
		Bip32Derivation: bip32Derivation,
	}
}

// deserialize attempts to recode a new POutput from the passed io.Reader.
func (po *POutput) deserialize(r io.Reader) error {
	for {
		keyint, keydata, err := getKey(r)
		if err != nil {
			return err
		}
		if keyint == -1 {
			// Reached separator byte
			break
		}

		value, err := wire.ReadVarBytes(
			r, 0, MaxPsbtValueLength, "PSBT value",
		)
		if err != nil {
			return err
		}

		switch OutputType(keyint) {

		case RedeemScriptOutputType:
			if po.RedeemScript != nil {
				return ErrDuplicateKey
			}
			if keydata != nil {
				return ErrInvalidKeydata
			}
			po.RedeemScript = value

		case WitnessScriptOutputType:
			if po.WitnessScript != nil {
				return ErrDuplicateKey
			}
			if keydata != nil {
				return ErrInvalidKeydata
			}
			po.WitnessScript = value

		case Bip32DerivationOutputType:
			if !validatePubkey(keydata) {
				return ErrInvalidKeydata
			}
			master, derivationPath, err := readBip32Derivation(value)
			if err != nil {
				return err
			}

			// Duplicate keys are not allowed
			for _, x := range po.Bip32Derivation {
				if bytes.Equal(x.PubKey, keydata) {
					return ErrDuplicateKey
				}
			}

			po.Bip32Derivation = append(po.Bip32Derivation,
				&Bip32Derivation{
					PubKey:               keydata,
					MasterKeyFingerprint: master,
					Bip32Path:            derivationPath,
				},
			)

		case TaprootInternalKeyOutputType:
			if po.TaprootInternalKey != nil {
				return ErrDuplicateKey
			}
			if keydata != nil {
				return ErrInvalidKeydata
			}

			if !validateXOnlyPubkey(value) {
				return ErrInvalidKeydata
			}

			po.TaprootInternalKey = value

		case TaprootTapTreeType:
			if po.TaprootTapTree != nil {
				return ErrDuplicateKey
			}
			if keydata != nil {
				return ErrInvalidKeydata
			}

			po.TaprootTapTree = value

		case TaprootBip32DerivationOutputType:
			if !validateXOnlyPubkey(keydata) {
				return ErrInvalidKeydata
			}

			taprootDerivation, err := readTaprootBip32Derivation(
				keydata, value,
			)
			if err != nil {
				return err
			}

			// Duplicate keys are not allowed.
			for _, x := range po.TaprootBip32Derivation {
				if bytes.Equal(x.XOnlyPubKey, keydata) {
					return ErrDuplicateKey
				}
			}

			po.TaprootBip32Derivation = append(
				po.TaprootBip32Derivation, taprootDerivation,
			)

		default:
			// Unknown type is allowed for inputs but not outputs.
			return ErrInvalidPsbtFormat
		}
	}

	return nil
}

// serialize attempts to write out the target POutput into the passed
// io.Writer.
func (po *POutput) serialize(w io.Writer) error {
	if po.RedeemScript != nil {
		err := serializeKVPairWithType(
			w, uint8(RedeemScriptOutputType), nil, po.RedeemScript,
		)
		if err != nil {
			return err
		}
	}
	if po.WitnessScript != nil {
		err := serializeKVPairWithType(
			w, uint8(WitnessScriptOutputType), nil, po.WitnessScript,
		)
		if err != nil {
			return err
		}
	}

	sort.Sort(Bip32Sorter(po.Bip32Derivation))
	for _, kd := range po.Bip32Derivation {
		err := serializeKVPairWithType(w,
			uint8(Bip32DerivationOutputType),
			kd.PubKey,
			SerializeBIP32Derivation(
				kd.MasterKeyFingerprint,
				kd.Bip32Path,
			),
		)
		if err != nil {
			return err
		}
	}

	if po.TaprootInternalKey != nil {
		err := serializeKVPairWithType(
			w, uint8(TaprootInternalKeyOutputType), nil,
			po.TaprootInternalKey,
		)
		if err != nil {
			return err
		}
	}

	if po.TaprootTapTree != nil {
		err := serializeKVPairWithType(
			w, uint8(TaprootTapTreeType), nil,
			po.TaprootTapTree,
		)
		if err != nil {
			return err
		}
	}

	sort.Slice(po.TaprootBip32Derivation, func(i, j int) bool {
		return po.TaprootBip32Derivation[i].SortBefore(
			po.TaprootBip32Derivation[j],
		)
	})
	for _, derivation := range po.TaprootBip32Derivation {
		value, err := serializeTaprootBip32Derivation(
			derivation,
		)
		if err != nil {
			return err
		}
		err = serializeKVPairWithType(
			w, uint8(TaprootBip32DerivationOutputType),
			derivation.XOnlyPubKey, value,
		)
		if err != nil {
			return err
		}
	}

	return nil
}