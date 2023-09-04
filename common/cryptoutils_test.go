/*
 *  Copyright IBM Corporation 2021, 2022
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 */

package common_test

import (
	"os"
	"testing"

	"github.com/konveyor/move2kube/common"
)

func TestEncryption(t *testing.T) {

	t.Run("RSA with Certificate", func(t *testing.T) {
		certificatePath := "testdata/testDataForCryptoutils/RSA_Cert.pem"
		certificateContent, err := os.ReadFile(certificatePath)
		if err != nil {
			t.Fatalf("Failed to read certificate file: %v", err)
		}
		data := "data_to_encrypt"
		encryptedData := common.EncryptRsaCertWrapper(string(certificateContent), data)

		if encryptedData == "" {
			t.Errorf("Expected encrypted data, but got an empty string")
		}
	})

	t.Run("AES with Pbkdf", func(t *testing.T) {
		key := "qwerty"
		data := "data_to_encrypt"

		encryptedData := common.EncryptAesCbcWithPbkdfWrapper(key, data)

		if encryptedData == "" {
			t.Errorf("Expected encrypted data, but got an empty string")
		}
	})

}
