package encryption

import (
	"encoding/base64"

	"github.com/pkg/errors"
	"github.com/s7techlab/cckit/convert"
	"github.com/s7techlab/cckit/router"
	"github.com/s7techlab/cckit/state"
)

var (
	// ErrKeyNotDefinedInTransientMap occurs when key not defined in transient map
	ErrKeyNotDefinedInTransientMap = errors.New(`encryption key is not defined in transient map`)
)

// State encrypting the data before putting to state and decrypting the data after getting from state
func State(c router.Context, key []byte) (state.State, error) {
	//current state
	s := c.State()

	s.UseKeyTransformer(KeyPartsEncryptedWith(key))
	s.UseStateGetTransformer(DecryptBytesWith(key))
	s.UseStatePutTransformer(EncryptBytesWith(key))

	return s, nil
}

// KeyFromTransient gets key for encrypting/decrypting from transient map
func KeyFromTransient(c router.Context) ([]byte, error) {
	tm, err := c.Stub().GetTransient()
	if err != nil {
		return nil, err
	}

	key, ok := tm[TransientMapKey]
	if !ok {
		return nil, ErrKeyNotDefinedInTransientMap
	}

	return key, nil
}

// StateWithTransientKey creates encrypted state state with provided key for symmetric encryption/decryption
func StateWithTransientKey(c router.Context) (state.State, error) {
	key, err := KeyFromTransient(c)
	if err != nil {
		return nil, err
	}
	return State(c, key)
}

// StateWithTransientKeyIfProvided creates encrypted state wrapper with provided key for symmetric encryption/decryption
// if key provided, otherwise - standard state wrapper without encryption
func StateWithTransientKeyIfProvided(c router.Context) (state.State, error) {
	key, err := KeyFromTransient(c)
	switch err {
	case nil:
		return State(c, key)
	case ErrKeyNotDefinedInTransientMap:
		//default state wrapper without encryption
		return c.State(), nil
	}
	return nil, err
}

// KeyPartsEncryptedWith encrypts key parts
func KeyPartsEncryptedWith(encryptKey []byte) state.KeyTransformer {
	return func(keyParts []string) ([]string, error) {
		keyPartsEnc := make([]string, len(keyParts))

		for i, p := range keyParts {
			keyPartEnc, err := Encrypt(encryptKey, p)
			if err != nil {
				return nil, errors.Wrap(err, `key part encrypt error`)
			}
			keyPartsEnc[i] = base64.StdEncoding.EncodeToString(keyPartEnc)
		}
		return keyPartsEnc, nil
	}
}

// DecryptBytesWith decrypts by with key - used for decrypting data after reading from state
func DecryptBytesWith(key []byte) state.FromBytesTransformer {
	return func(bb []byte, config ...interface{}) (interface{}, error) {
		decrypted, err := Decrypt(key, bb)
		if err != nil {
			return nil, errors.Wrap(err, `decrypt bytes`)
		}
		if len(config) == 0 {
			return decrypted, nil
		}
		return convert.FromBytes(decrypted, config[0])
	}
}

// EncryptBytesWith encrypts bytes with key - used for encrypting data for state
func EncryptBytesWith(key []byte) state.ToBytesTransformer {
	return func(v interface{}, config ...interface{}) ([]byte, error) {
		bb, err := convert.ToBytes(v)
		if err != nil {
			return nil, err
		}
		return Encrypt(key, bb)
	}
}
