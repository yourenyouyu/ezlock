package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"io"
	"net/http"
)

// 返回正常响应
func ResponseOk(result interface{}, c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":   OK,
		"msg":    ERR_MSG_MAP[OK],
		"result": result,
	})
}
func ResponseOkWithCount(count int, result interface{}, c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":   OK,
		"msg":    ERR_MSG_MAP[OK],
		"count":  count,
		"result": result,
	})
}

// 返回错误响应
func ResponseError(code int, msg string, c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":   code,
		"msg":    fmt.Sprintf("%s: %s", ERR_MSG_MAP[code], msg),
		"result": "",
	})
}

// 校验参数
func CheckParam(params interface{}, c *gin.Context) (ok bool) {
	ok = true
	if err := c.ShouldBindWith(params, binding.Default(c.Request.Method, c.ContentType())); err != nil {
		ok = false
		msg := fmt.Sprintf("参数验证失败: [ %s ]", err.Error())
		ResponseError(PARAM_ERR, msg, c)
	}
	return
}

// 使用PKCS7进行填充
func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS7UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

// iv 要是16位
func AesCBCEncrypt(iv, rawData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//填充原文
	blockSize := block.BlockSize()
	rawData = PKCS7Padding(rawData, blockSize)
	//初始向量IV必须是唯一，但不需要保密
	cipherText := make([]byte, blockSize+len(rawData))
	//block大小 16
	//iv := cipherText[:blockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	//block大小和初始向量大小一定要一致
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(cipherText[blockSize:], rawData)
	return cipherText, nil
}

// 因为加密后有些字符不可见 进行base64的编码响应
func Encrypt(iv, rawData, key []byte) (string, error) {
	data, err := AesCBCEncrypt(iv, rawData, key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func AesCBCDncrypt(encryptData, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()

	if len(encryptData) < blockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := encryptData[:blockSize]
	encryptData = encryptData[blockSize:]

	// CBC mode always works in whole blocks.
	if len(encryptData)%blockSize != 0 {
		return nil, errors.New("ciphertext is not a multiple of the block size")
	}

	mode := cipher.NewCBCDecrypter(block, iv)

	mode.CryptBlocks(encryptData, encryptData)
	//解填充
	encryptData = PKCS7UnPadding(encryptData)
	return encryptData, nil
}

func Dncrypt(rawData string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(rawData)
	if err != nil {
		return "", err
	}
	dnData, err := AesCBCDncrypt(data, key)
	if err != nil {
		return "", err
	}
	return string(dnData), nil
}
