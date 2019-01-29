// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
////////////////////////////////////////////////////////////////////////////////

package deterministicaead_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/google/tink/go/deterministicaead"
	"github.com/google/tink/go/subtle/random"
	"github.com/google/tink/go/tink"

	subtedeterministicaead "github.com/google/tink/go/subtle/deterministicaead"
	aspb "github.com/google/tink/proto/aes_siv_go_proto"
	tinkpb "github.com/google/tink/proto/tink_go_proto"
)

func TestAESSIVPrimitive(t *testing.T) {
	km, err := tink.GetKeyManager(deterministicaead.AESSIVTypeURL)
	if err != nil {
		t.Fatalf("cannot obtain AESSIV key manager: %s", err)
	}
	m, err := km.NewKey(nil)
	if err != nil {
		t.Errorf("km.NewKey(nil) = _, %v; want _, nil", err)
	}
	key, _ := m.(*aspb.AesSivKey)
	serializedKey, _ := proto.Marshal(key)
	p, err := km.Primitive(serializedKey)
	if err != nil {
		t.Errorf("km.Primitive(%v) = %v; want nil", serializedKey, err)
	}
	if err := validateAESSIVPrimitive(p, key); err != nil {
		t.Errorf("validateAESSIVPrimitive(p, key) = %v; want nil", err)
	}
}

func TestAESSIVPrimitiveWithInvalidKeys(t *testing.T) {
	km, err := tink.GetKeyManager(deterministicaead.AESSIVTypeURL)
	if err != nil {
		t.Errorf("cannot obtain AESSIV key manager: %s", err)
	}
	invalidKeys := genInvalidAESSIVKeys()
	for _, key := range invalidKeys {
		serializedKey, _ := proto.Marshal(key)
		if _, err := km.Primitive(serializedKey); err == nil {
			t.Errorf("km.Primitive(%v) = _, nil; want _, err", serializedKey)
		}
	}
}

func TestAESSIVNewKey(t *testing.T) {
	km, err := tink.GetKeyManager(deterministicaead.AESSIVTypeURL)
	if err != nil {
		t.Errorf("cannot obtain AESSIV key manager: %s", err)
	}
	m, err := km.NewKey(nil)
	if err != nil {
		t.Errorf("km.NewKey(nil) = _, %v; want _, nil", err)
	}
	key, _ := m.(*aspb.AesSivKey)
	if err := validateAESSIVKey(key); err != nil {
		t.Errorf("validateAESSIVKey(%v) = %v; want nil", key, err)
	}
}

func TestAESSIVNewKeyData(t *testing.T) {
	km, err := tink.GetKeyManager(deterministicaead.AESSIVTypeURL)
	if err != nil {
		t.Errorf("cannot obtain AESSIV key manager: %s", err)
	}
	kd, err := km.NewKeyData(nil)
	if err != nil {
		t.Errorf("km.NewKeyData(nil) = _, %v; want _, nil", err)
	}
	if kd.TypeUrl != deterministicaead.AESSIVTypeURL {
		t.Errorf("TypeUrl: %v != %v", kd.TypeUrl, deterministicaead.AESSIVTypeURL)
	}
	if kd.KeyMaterialType != tinkpb.KeyData_SYMMETRIC {
		t.Errorf("KeyMaterialType: %v != SYMMETRIC", kd.KeyMaterialType)
	}
	key := new(aspb.AesSivKey)
	if err := proto.Unmarshal(kd.Value, key); err != nil {
		t.Errorf("proto.Unmarshal(%v, key) = %v; want nil", kd.Value, err)
	}
	if err := validateAESSIVKey(key); err != nil {
		t.Errorf("validateAESSIVKey(%v) = %v; want nil", key, err)
	}
}

func TestAESSIVDoesSupport(t *testing.T) {
	km, err := tink.GetKeyManager(deterministicaead.AESSIVTypeURL)
	if err != nil {
		t.Errorf("cannot obtain AESSIV key manager: %s", err)
	}
	if !km.DoesSupport(deterministicaead.AESSIVTypeURL) {
		t.Errorf("AESSIVKeyManager must support %s", deterministicaead.AESSIVTypeURL)
	}
	if km.DoesSupport("some bad type") {
		t.Errorf("AESSIVKeyManager must only support %s", deterministicaead.AESSIVTypeURL)
	}
}

func TestAESSIVTypeURL(t *testing.T) {
	km, err := tink.GetKeyManager(deterministicaead.AESSIVTypeURL)
	if err != nil {
		t.Errorf("cannot obtain AESSIV key manager: %s", err)
	}
	if kt := km.TypeURL(); kt != deterministicaead.AESSIVTypeURL {
		t.Errorf("km.TypeURL() = %s; want %s", kt, deterministicaead.AESSIVTypeURL)
	}
}

func validateAESSIVPrimitive(p interface{}, key *aspb.AesSivKey) error {
	cipher := p.(*subtedeterministicaead.AESSIV)
	// try to encrypt and decrypt
	pt := random.GetRandomBytes(32)
	aad := random.GetRandomBytes(32)
	ct, err := cipher.EncryptDeterministically(pt, aad)
	if err != nil {
		return fmt.Errorf("encryption failed")
	}
	decrypted, err := cipher.DecryptDeterministically(ct, aad)
	if err != nil {
		return fmt.Errorf("decryption failed")
	}
	if !bytes.Equal(decrypted, pt) {
		return fmt.Errorf("decryption failed")
	}
	return nil
}

func validateAESSIVKey(key *aspb.AesSivKey) error {
	if key.Version != deterministicaead.AESSIVKeyVersion {
		return fmt.Errorf("incorrect key version: keyVersion != %d", deterministicaead.AESSIVKeyVersion)
	}
	if uint32(len(key.KeyValue)) != subtedeterministicaead.AESSIVKeySize {
		return fmt.Errorf("incorrect key size: keySize != %d", subtedeterministicaead.AESSIVKeySize)
	}

	// Try to encrypt and decrypt.
	p, err := subtedeterministicaead.NewAESSIV(key.KeyValue)
	if err != nil {
		return fmt.Errorf("invalid key: %v", key.KeyValue)
	}
	return validateAESSIVPrimitive(p, key)
}

func genInvalidAESSIVKeys() []*aspb.AesSivKey {
	return []*aspb.AesSivKey{
		// Bad key size.
		&aspb.AesSivKey{
			Version:  deterministicaead.AESSIVKeyVersion,
			KeyValue: random.GetRandomBytes(16),
		},
		&aspb.AesSivKey{
			Version:  deterministicaead.AESSIVKeyVersion,
			KeyValue: random.GetRandomBytes(32),
		},
		&aspb.AesSivKey{
			Version:  deterministicaead.AESSIVKeyVersion,
			KeyValue: random.GetRandomBytes(63),
		},
		&aspb.AesSivKey{
			Version:  deterministicaead.AESSIVKeyVersion,
			KeyValue: random.GetRandomBytes(65),
		},
		// Bad version.
		&aspb.AesSivKey{
			Version:  deterministicaead.AESSIVKeyVersion + 1,
			KeyValue: random.GetRandomBytes(subtedeterministicaead.AESSIVKeySize),
		},
	}
}
