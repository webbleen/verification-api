package services

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

// SignatureVerifier App Store 签名验证器
type SignatureVerifier struct {
	certCache      map[string]*x509.Certificate
	mutex          sync.RWMutex
	lastCertUpdate time.Time
	certCacheTTL   time.Duration
}

// NewSignatureVerifier 创建新的签名验证器
func NewSignatureVerifier() *SignatureVerifier {
	return &SignatureVerifier{
		certCache:    make(map[string]*x509.Certificate),
		certCacheTTL: time.Hour * 24, // 证书缓存24小时
	}
}

// SignatureInfo 签名信息
type SignatureInfo struct {
	CertificateChain []string `json:"x5c"`
	Timestamp        int64    `json:"timestamp"`
	Signature        string   `json:"signature"`
}

// VerifyNotification 验证 App Store 通知签名
func (v *SignatureVerifier) VerifyNotification(notificationBody []byte, signatureHeader string) error {
	if signatureHeader == "" {
		return fmt.Errorf("missing X-Apple-Notification-Signature header")
	}

	// 提取签名信息
	signature, err := v.extractSignature(signatureHeader)
	if err != nil {
		return fmt.Errorf("failed to extract signature: %w", err)
	}

	// 获取证书链
	certChain, err := v.getCertificateChain(signature.CertificateChain)
	if err != nil {
		return fmt.Errorf("failed to get certificate chain: %w", err)
	}

	// 验证证书链
	if err := v.verifyCertificateChain(certChain); err != nil {
		return fmt.Errorf("failed to verify certificate chain: %w", err)
	}

	// 验证签名
	if err := v.verifySignature(notificationBody, signature, certChain[0]); err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	// 验证时间戳
	if err := v.verifyTimestamp(signature.Timestamp); err != nil {
		return fmt.Errorf("failed to verify timestamp: %w", err)
	}

	return nil
}

// extractSignature 从请求头中提取签名信息
func (v *SignatureVerifier) extractSignature(signatureHeader string) (*SignatureInfo, error) {
	// 解码 base64 签名
	signatureData, err := base64.StdEncoding.DecodeString(signatureHeader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// 解析签名信息
	var signatureInfo SignatureInfo
	if err := json.Unmarshal(signatureData, &signatureInfo); err != nil {
		return nil, fmt.Errorf("failed to unmarshal signature info: %w", err)
	}

	return &signatureInfo, nil
}

// getCertificateChain 获取证书链
func (v *SignatureVerifier) getCertificateChain(certChain []string) ([]*x509.Certificate, error) {
	var certificates []*x509.Certificate

	for _, certPEM := range certChain {
		// 检查缓存
		v.mutex.RLock()
		if cert, exists := v.certCache[certPEM]; exists {
			certificates = append(certificates, cert)
			v.mutex.RUnlock()
			continue
		}
		v.mutex.RUnlock()

		// 解析证书
		cert, err := v.parseCertificate(certPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}

		// 缓存证书
		v.mutex.Lock()
		v.certCache[certPEM] = cert
		v.mutex.Unlock()

		certificates = append(certificates, cert)
	}

	return certificates, nil
}

// parseCertificate 解析 PEM 格式的证书
func (v *SignatureVerifier) parseCertificate(certPEM string) (*x509.Certificate, error) {
	// 确保证书格式正确（添加 PEM 头尾）
	if !strings.HasPrefix(certPEM, "-----BEGIN CERTIFICATE-----") {
		certPEM = "-----BEGIN CERTIFICATE-----\n" + certPEM + "\n-----END CERTIFICATE-----"
	}

	block, _ := pem.Decode([]byte(certPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return cert, nil
}

// verifyCertificateChain 验证证书链
func (v *SignatureVerifier) verifyCertificateChain(certChain []*x509.Certificate) error {
	if len(certChain) == 0 {
		return fmt.Errorf("empty certificate chain")
	}

	// 验证证书链中的每个证书
	for i, cert := range certChain {
		// 检查证书是否过期
		now := time.Now()
		if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
			return fmt.Errorf("certificate %d is expired or not yet valid", i)
		}

		// 如果不是根证书，验证签名
		if i > 0 {
			parentCert := certChain[i-1]
			if err := cert.CheckSignatureFrom(parentCert); err != nil {
				return fmt.Errorf("certificate %d signature verification failed: %w", i, err)
			}
		}
	}

	// 验证根证书是否为苹果证书
	rootCert := certChain[len(certChain)-1]
	if !v.isAppleRootCertificate(rootCert) {
		return fmt.Errorf("invalid root certificate: not from Apple")
	}

	return nil
}

// isAppleRootCertificate 检查是否为苹果根证书
func (v *SignatureVerifier) isAppleRootCertificate(cert *x509.Certificate) bool {
	// 检查证书的 Subject 和 Issuer
	appleSubjects := []string{
		"Apple Root CA",
		"Apple Inc.",
		"Apple Computer, Inc.",
	}

	for _, subject := range appleSubjects {
		if strings.Contains(cert.Subject.String(), subject) {
			return true
		}
	}

	return false
}

// verifySignature 验证签名
func (v *SignatureVerifier) verifySignature(notificationBody []byte, signature *SignatureInfo, cert *x509.Certificate) error {
	// 解码签名
	signatureBytes, err := base64.StdEncoding.DecodeString(signature.Signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// 创建签名数据
	signatureData := v.createSignatureData(notificationBody, signature.Timestamp)

	// 计算哈希
	hash := sha256.Sum256(signatureData)

	// 验证 ECDSA 签名
	publicKey, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate does not contain ECDSA public key")
	}

	// 解析签名
	var r, s []byte
	if len(signatureBytes) != 64 {
		return fmt.Errorf("invalid signature length: expected 64, got %d", len(signatureBytes))
	}
	r = signatureBytes[:32]
	s = signatureBytes[32:]

	// 将字节转换为 big.Int
	rBig := new(big.Int).SetBytes(r)
	sBig := new(big.Int).SetBytes(s)

	// 验证签名
	if !ecdsa.Verify(publicKey, hash[:], rBig, sBig) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// createSignatureData 创建签名数据
func (v *SignatureVerifier) createSignatureData(notificationBody []byte, timestamp int64) []byte {
	// 签名数据格式：timestamp + "." + notification_body
	timestampStr := fmt.Sprintf("%d", timestamp)
	return []byte(timestampStr + "." + string(notificationBody))
}

// verifyTimestamp 验证时间戳
func (v *SignatureVerifier) verifyTimestamp(timestamp int64) error {
	now := time.Now().Unix()

	// 允许5分钟的时间差
	timeDiff := now - timestamp
	if timeDiff < -300 || timeDiff > 300 {
		return fmt.Errorf("timestamp is too old or too far in the future: %d seconds difference", timeDiff)
	}

	return nil
}

// ClearCache 清除证书缓存
func (v *SignatureVerifier) ClearCache() {
	v.mutex.Lock()
	defer v.mutex.Unlock()

	v.certCache = make(map[string]*x509.Certificate)
	v.lastCertUpdate = time.Time{}
}

// IsCacheValid 检查缓存是否有效
func (v *SignatureVerifier) IsCacheValid() bool {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return time.Since(v.lastCertUpdate) < v.certCacheTTL
}

