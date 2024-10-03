package identitydoc

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"time"
)

// ErrInvalidDocument represents the failure when the document is not verified
// by the signature
var ErrInvalidDocument = errors.New("The provided identify document does not match the signature")

// ErrUnknownRegion indicates no certificate was found for the given region
var ErrUnknownRegion = errors.New("Certificate not found for the provided region")

const genericAWSPublicCertificateIdentity = `-----BEGIN CERTIFICATE-----
MIIDIjCCAougAwIBAgIJAKnL4UEDMN/FMA0GCSqGSIb3DQEBBQUAMGoxCzAJBgNV
BAYTAlVTMRMwEQYDVQQIEwpXYXNoaW5ndG9uMRAwDgYDVQQHEwdTZWF0dGxlMRgw
FgYDVQQKEw9BbWF6b24uY29tIEluYy4xGjAYBgNVBAMTEWVjMi5hbWF6b25hd3Mu
Y29tMB4XDTE0MDYwNTE0MjgwMloXDTI0MDYwNTE0MjgwMlowajELMAkGA1UEBhMC
VVMxEzARBgNVBAgTCldhc2hpbmd0b24xEDAOBgNVBAcTB1NlYXR0bGUxGDAWBgNV
BAoTD0FtYXpvbi5jb20gSW5jLjEaMBgGA1UEAxMRZWMyLmFtYXpvbmF3cy5jb20w
gZ8wDQYJKoZIhvcNAQEBBQADgY0AMIGJAoGBAIe9GN//SRK2knbjySG0ho3yqQM3
e2TDhWO8D2e8+XZqck754gFSo99AbT2RmXClambI7xsYHZFapbELC4H91ycihvrD
jbST1ZjkLQgga0NE1q43eS68ZeTDccScXQSNivSlzJZS8HJZjgqzBlXjZftjtdJL
XeE4hwvo0sD4f3j9AgMBAAGjgc8wgcwwHQYDVR0OBBYEFCXWzAgVyrbwnFncFFIs
77VBdlE4MIGcBgNVHSMEgZQwgZGAFCXWzAgVyrbwnFncFFIs77VBdlE4oW6kbDBq
MQswCQYDVQQGEwJVUzETMBEGA1UECBMKV2FzaGluZ3RvbjEQMA4GA1UEBxMHU2Vh
dHRsZTEYMBYGA1UEChMPQW1hem9uLmNvbSBJbmMuMRowGAYDVQQDExFlYzIuYW1h
em9uYXdzLmNvbYIJAKnL4UEDMN/FMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQEF
BQADgYEAFYcz1OgEhQBXIwIdsgCOS8vEtiJYF+j9uO6jz7VOmJqO+pRlAbRlvY8T
C1haGgSI/A1uZUKs/Zfnph0oEI0/hu1IIJ/SKBDtN5lvmZ/IzbOPIJWirlsllQIQ
7zvWbGd9c9+Rm3p04oTvhup99la7kZqevJK0QRdD/6NpCKsqP/0=
-----END CERTIFICATE-----`

var (
	awsCert *x509.Certificate

	awsCertPEM, _ = pem.Decode([]byte(genericAWSPublicCertificateIdentity))
)

func init() {
	var err error
	if awsCert, err = x509.ParseCertificate(awsCertPEM.Bytes); err != nil {
		// We are loading static data, if something goes wrong here it's a real
		// problem
		panic(err)
	}
}

// InstanceIdentityDocument represents the information contained in an instances
// identity document
// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-instance-metadata.html
type InstanceIdentityDocument struct {
	InstanceID       string    `json:"instanceId"`
	AccountID        string    `json:"accountId"`
	PrivateIP        string    `json:"privateIp"`
	Region           string    `json:"region"`
	AvailabilityZone string    `json:"availabilityZone"`
	PendingTime      time.Time `json:"pendingTime"`
	InstanceType     string    `json:"instanceType"`
	ImageID          string    `json:"imageId"`

	Doc json.RawMessage `json:"-"`
	Sig []byte          `json:"-"`
}

// VerifyDocumentAndSignature will confirm that the document is correct by
// validating it against the signature and cert for the given region. It will
// return the parsed document if it's valid, or ErrInvalidDocument if it's not.
// Document is the data returned from:
// http://169.254.169.254/latest/dynamic/instance-identity/document
// Signature is returned from:
// http://169.254.169.254/latest/dynamic/instance-identity/signature
// If the region is unknown or has no cert, ErrUnknownRegion region will be
// returned. If there are any other errors, the error will be passed on.
func VerifyDocumentAndSignature(region string, document, signature []byte) (*InstanceIdentityDocument, error) {
	rawSig, err := base64.StdEncoding.DecodeString(string(signature))
	if err != nil {
		return nil, err
	}

	iid := &InstanceIdentityDocument{
		Doc: document,
		Sig: rawSig,
	}

	if err := json.Unmarshal(document, iid); err != nil {
		return nil, ErrInvalidDocument
	}

	return iid, iid.CheckSignature()
}

func (d InstanceIdentityDocument) CheckSignature() error {
	return awsCert.CheckSignature(x509.SHA256WithRSA, d.Doc, d.Sig)
}
