/*
 *  Copyright IBM Corporation 2021
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

package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/pbkdf2"
)

// EncryptRsaCertWrapper can be used to encrypt the data using RSA PKCS1v15 algorithm with certificate as key
func EncryptRsaCertWrapper(certificate string, data string) string {

	rsaPublicKey, err := getRSAPublicKeyFromCertificate([]byte(certificate))
	if err != nil {
		logrus.Errorf("failed to parse the certificate and get the RSA public key from it. Error: %q", err)
	}

	out, err := rsaEncrypt(rsaPublicKey, []byte(data))
	if err != nil {
		logrus.Errorf("failed to encrypt the data with RSA PKCS1v15 algorithm with certificate as key : %q", err)
	}

	return string(out)
}

// EncryptAesCbcWithPbkdfWrapper can be used to encrypt the data using AES 256 CBC mode with Pbkdf key derivation
func EncryptAesCbcWithPbkdfWrapper(key string, data string) string {
	salt := make([]byte, 16)
	_, err := rand.Read(salt)
	if err != nil {
		logrus.Errorf("failed to prepare the salt : %s", err)
	}
	out, err := aesCbcEncryptWithPbkdf([]byte(key), salt, []byte(data))
	if err != nil {
		logrus.Errorf("failed to encrypt the data using AES 256 CBC mode with Pbkdf key derivation : %q", err)
	}
	opensslOut := toOpenSSLFormat(salt, out)
	return string(opensslOut)
}

// getRSAPublicKeyFromCertificate gets a RSA public key from a PEM-encoded certificate.crt file.
func getRSAPublicKeyFromCertificate(certificateInPemFormat []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(certificateInPemFormat)
	if block == nil {
		return nil, fmt.Errorf("invalid certificate. Expected a PEM encoded certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the x509 certificate. Error: %q", err)
	}
	pubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to type cast the public key from the x509 certificate. Actual type: %T and value: %+v", cert.PublicKey, cert.PublicKey)
	}
	return pubKey, nil
}

// rsaEncrypt encrypts the plain text using the RSA public key.
func rsaEncrypt(publicKey *rsa.PublicKey, plainText []byte) ([]byte, error) {
	cipherText, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, plainText)
	if err != nil {
		return cipherText, fmt.Errorf("failed to RSA encrypt the plain text. Error: %q", err)
	}
	return cipherText, nil
}

// deriveAesKeyAndIv returns an AES key and IV derived from the password and salt using PBKDF2 function.
func deriveAesKeyAndIv(password, salt []byte) ([]byte, []byte) {
	aesKeyAndIv := pbkdf2.Key(password, salt, 10000, 32+16, sha256.New)
	return aesKeyAndIv[:32], aesKeyAndIv[32:]
}

// aesCbcEncryptWithPbkdf derives an AES key and IV using the given password and salt and then encrypts the plain text.
func aesCbcEncryptWithPbkdf(password, salt, plainText []byte) ([]byte, error) {

	// derive an AES key and IV using the password and the salt

	aesKey, iv := deriveAesKeyAndIv(password, salt)

	// encrypt the workload using the AES key

	aesCipher, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new AES cipher using the key. Error: %q", err)
	}

	// pad the plain text as per PKCS#5 https://en.wikipedia.org/wiki/Padding_(cryptography)#PKCS#5_and_PKCS#7

	paddingRequired := aes.BlockSize - len(plainText)%aes.BlockSize
	if paddingRequired == 0 {
		paddingRequired = aes.BlockSize
	}
	padding := make([]byte, paddingRequired)
	for i := 0; i < paddingRequired; i++ {
		padding[i] = byte(paddingRequired)
	}

	paddedPlainText := append(plainText, padding...)
	cipherText := make([]byte, len(paddedPlainText))

	aesCbcEncrypter := cipher.NewCBCEncrypter(aesCipher, iv)
	aesCbcEncrypter.CryptBlocks(cipherText, paddedPlainText)
	return cipherText, nil
}

// toOpenSSLFormat converts the cipher text to OpenSSL encrypted with password and salt format.
// http://justsolve.archiveteam.org/wiki/OpenSSL_salted_format
func toOpenSSLFormat(salt, cipherText []byte) []byte {
	return append(append([]byte("Salted__"), salt...), cipherText...)
}
