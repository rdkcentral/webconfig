/**
* Copyright 2021 Comcast Cable Communications Management, LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
* SPDX-License-Identifier: Apache-2.0
*/
/*
 * Some code in Encrypt/Decrypt is:
 * Copyright 2012 The Go Authors. All rights reserved.
 * Licensed under the BSD-3 License
 */
package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/rdkcentral/webconfig/common"
	"github.com/go-akka/configuration"
	log "github.com/sirupsen/logrus"
)

/*
## DESCRIPTION
% ENCRYPTION:

we first generate a sha1 digest

digest = digest(iv, plaintext)

padded_core_data = padding(digest + plaintext)

then we encrypt this

encrypted = encode(digest + plaintext)

return b64encode( iv + encrypted )

% DECRYPTION:

(iv + encrypted = b64decode(input)

raw_decoded = decode(iv, encrypted)

unpadded_core_data = unpadding(raw_decoded)

remove the first 20 bytes, the "digest" part ==> plaintext
*/

type AesCodec struct {
	key []byte
}

const (
	envNameDefault = "WEBCONFIG_KEY"
)

// for controlled testing only
// var staticIv []byte{111, 114, 219, 23, 120, 151, 157, 32, 117, 31, 98, 99, 106, 3, 169, 224}

func NewAesCodec(conf *configuration.Config, args ...string) (*AesCodec, error) {
	envName := conf.GetString("webconfig.security.encryption_key_env_name", envNameDefault)

	var defaultCodec AesCodec

	var enckeyB64 string
	if len(args) > 0 {
		enckeyB64 = args[0]
	} else {
		enckeyB64 = os.Getenv(envName)
	}

	if len(enckeyB64) == 0 {
		err := fmt.Errorf("No env %v", envName)
		return &defaultCodec, common.NewError(err)
	}

	key, err := base64.StdEncoding.DecodeString(enckeyB64)
	if err != nil {
		return &defaultCodec, common.NewError(err)
	}

	return &AesCodec{
		key: key,
	}, nil
}

func (c *AesCodec) Decrypt(encryptedB64 string) (string, error) {
	// CBC decryption
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedB64)
	if err != nil {
		return "", err
	}

	if c.key == nil {
		return string(ciphertext), nil
	}

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	if len(ciphertext) < aes.BlockSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// CBC mode always works in whole blocks.
	if len(ciphertext)%aes.BlockSize != 0 {
		return "", fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	// CryptBlocks can work in-place if the two arguments are the same.
	mode.CryptBlocks(ciphertext, ciphertext)

	// If the original plaintext lengths are not a multiple of the block
	// size, padding would have to be added when encrypting, which would be
	// removed at this point. For an example, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. However, it's
	// critical to note that ciphertexts must be authenticated (i.e. by
	// using crypto/hmac) before being decrypted in order to avoid creating
	// a padding oracle.

	// unpadding
	index := len(ciphertext) - 1

	for {
		if ciphertext[index] == '\x00' || ciphertext[index] == '\x80' {
			index--
		} else {
			break
		}
	}

	if index < 20 {
		return "", fmt.Errorf("decrypt error")
	}

	decrypted := ciphertext[20 : index+1]
	return string(decrypted), nil
}

func Digest(iv []byte, plaintextstr string) []byte {
	buffer := bytes.NewBuffer(iv)
	buffer.WriteString(plaintextstr)
	h := sha1.New()
	h.Write(buffer.Bytes())
	bs := h.Sum(nil)
	return bs
}

func (c *AesCodec) Encrypt(plaintextstr string) (string, error) {
	if c.key == nil {
		return base64.StdEncoding.EncodeToString([]byte(plaintextstr)), nil
	}
	// Load your secret key from a safe place and reuse it across multiple
	// NewCipher calls. (Obviously don't use this example key for anything
	// real.) If you want to convert a passphrase to a key, use a suitable
	// package like bcrypt or scrypt.
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	hashed := Digest(iv, plaintextstr)
	buffer := bytes.NewBuffer(hashed)
	buffer.WriteString(plaintextstr)
	hashedIvPlain := buffer.Bytes()

	// CBC mode works on blocks so plaintexts may need to be padded to the
	// next whole block. For an example of such padding, see
	// https://tools.ietf.org/html/rfc5246#section-6.2.3.2. Here we'll
	// assume that the plaintext is already of the correct length.
	if len(hashedIvPlain)%aes.BlockSize != 0 {
		// panic("plaintext is not a multiple of the block size")
		hashedIvPlain = Padding(hashedIvPlain, block.BlockSize())
	}

	// The IV needs to be unique, but not secure. Therefore it's common to
	// include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(hashedIvPlain))
	civ := ciphertext[:aes.BlockSize]
	copy(civ, iv)

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], hashedIvPlain)

	// It's important to remember that ciphertexts must be authenticated
	// (i.e. by using crypto/hmac) as well as being encrypted in order to
	// be secure.

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize

	if padding == 0 {
		return ciphertext
	} else if padding == 1 {
		return append(ciphertext, byte('\x80'))
	} else {
		padtext := []byte{'\x80'}
		padtext = append(padtext, bytes.Repeat([]byte{'\x00'}, padding-1)...)
		return append(ciphertext, padtext...)
	}
}

func (c *AesCodec) DecryptBytes(encbytes []byte) ([]byte, error) {
	var ciphertext []byte

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return ciphertext, err
	}

	ciphertext = make([]byte, len(encbytes))
	copy(ciphertext, encbytes)

	if len(ciphertext) < aes.BlockSize {
		return ciphertext, fmt.Errorf("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	if len(ciphertext)%aes.BlockSize != 0 {
		return ciphertext, fmt.Errorf("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	mode.CryptBlocks(ciphertext, ciphertext)

	// unpadding
	index := len(ciphertext) - 1

	for {
		if ciphertext[index] == '\x00' || ciphertext[index] == '\x80' {
			index--
		} else {
			break
		}
	}

	if index < 20 {
		return ciphertext, fmt.Errorf("decrypt error")
	}

	decrypted := ciphertext[20 : index+1]
	return decrypted, nil
}

func DigestBytes(iv []byte, plainbytes []byte) []byte {
	buffer := bytes.NewBuffer(iv)
	buffer.Write(plainbytes)
	h := sha1.New()
	h.Write(buffer.Bytes())
	bs := h.Sum(nil)
	return bs
}

func (c *AesCodec) EncryptBytes(plainbytes []byte) ([]byte, error) {
	var ciphertext []byte

	block, err := aes.NewCipher(c.key)
	if err != nil {
		return ciphertext, err
	}

	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return ciphertext, err
	}

	hashed := DigestBytes(iv, plainbytes)
	buffer := bytes.NewBuffer(hashed)
	buffer.Write(plainbytes)
	hashedIvPlain := buffer.Bytes()

	if len(hashedIvPlain)%aes.BlockSize != 0 {
		hashedIvPlain = Padding(hashedIvPlain, block.BlockSize())
	}

	ciphertext = make([]byte, aes.BlockSize+len(hashedIvPlain))
	civ := ciphertext[:aes.BlockSize]
	copy(civ, iv)

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], hashedIvPlain)

	return ciphertext, nil
}

func (c *AesCodec) LogResponseDebug(fields log.Fields, bbytes []byte) {
	encbytes, err := c.EncryptBytes(bbytes)
	if err != nil {
		log.WithFields(fields).Error(err.Error())
		return
	}

	response := base64.StdEncoding.EncodeToString(encbytes)
	fields["response"] = response
	log.WithFields(fields).Debug("")
}

// in base64 format
func GetRandomEncryptionKey() string {
	// use 16-byte for simplicity
	bbytes := make([]byte, 16)
	rand.Read(bbytes)
	return base64.StdEncoding.EncodeToString(bbytes)
}

// test codec shared by other modules
var (
	testCodec *AesCodec
)
