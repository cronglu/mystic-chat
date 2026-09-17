package network

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"io"
	"math/big"
	"time"
)

// DefaultLobbySalt 用于公开大厅防抓包的默认底座混淆盐
const DefaultLobbySalt = "VANISH_WUXIA_LOBBY_DEFENSE_SALT_2026"

// DeriveRoomKey 通过江湖暗号（房间口令）派生出 AES-256 密钥
// 即使未输入暗号（公开大厅），也默认使用底层混淆密钥进行全量对称加密，防止路由器抓包明文
func DeriveRoomKey(secret string) []byte {
	if len(secret) == 0 {
		// 默认大厅混淆密钥：杜绝任何未加密的明文在局域网网关上裸奔
		hash := sha256.Sum256([]byte(DefaultLobbySalt))
		return hash[:]
	}
	hash := sha256.Sum256([]byte("VANISH_ROOM_SALT:" + secret))
	return hash[:]
}

// EncryptPayload 使用 AES-256-GCM 对消息内容执行端到端加密
func EncryptPayload(key []byte, plaintext []byte) ([]byte, error) {
	if len(key) == 0 {
		key = DeriveRoomKey("")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptPayload 使用 AES-256-GCM 解密消息密文
func DecryptPayload(key []byte, ciphertext []byte) ([]byte, error) {
	if len(key) == 0 {
		key = DeriveRoomKey("")
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("密文长度不足，无法解析 nonce")
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, errors.New("江湖暗号不匹配或密文已被篡改/路由器截断")
	}

	return plaintext, nil
}

// DeriveAdminTokenHash 计算管理员紧急熔断令旗的校验哈希
func DeriveAdminTokenHash(adminKey string) [32]byte {
	return sha256.Sum256([]byte("VANISH_ADMIN_MELT_TOKEN:" + adminKey))
}

// GenerateEphemeralTLSConfig 在内存中动态生成自签名临时 TLS 1.3 证书与配置
// 所有 TCP P2P 流量均被 TLS 1.3 强加密隧道包裹，路由器/交换机抓包只能看到纯二进制 TLS 流量，无法识别协议与任何特征
func GenerateEphemeralTLSConfig() (*tls.Config, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		serialNumber = big.NewInt(time.Now().UnixNano())
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   "vanish-p2p-node",
			Organization: []string{"Vanish Jianghu"},
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(24 * 365 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, err
	}

	tlsCert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  priv,
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		InsecureSkipVerify: true, // P2P 对等节点使用自签名传输层加密隧道
		MinVersion:         tls.VersionTLS13,
	}, nil
}
