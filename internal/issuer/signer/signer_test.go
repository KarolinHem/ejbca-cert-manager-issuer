package signer

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestEjbcaHealthCheckerFromIssuerAndSecretData(t *testing.T) {

}

// EJBCA_CA_NAME=ManagementCA ;EJBCA_CERTIFICATE_PROFILE_NAME=tlsServerAuth;EJBCA_CLIENT_CERT_PATH=C:\Users\hroszell\OneDrive - Keyfactor\Documents\coding\EJBCA\certs\adminHaydenRoszell.pem;EJBCA_END_ENTITY_PROFILE_NAME=haydenEndEntity;EJBCA_HOSTNAME=https://it-ca01.pkihosted-dev.c2company.com/ejbca/

func TestEjbcaSignerFromIssuerAndSecretData(t *testing.T) {
	pathToClientCert := os.Getenv("EJBCA_CLIENT_CERT_PATH")
	hostname := os.Getenv("EJBCA_HOSTNAME")
	ejbcaCaName := os.Getenv("EJBCA_CA_NAME")
	ejbcaCertificateProfileName := os.Getenv("EJBCA_CERTIFICATE_PROFILE_NAME")
	ejbcaEndEntityProfileName := os.Getenv("EJBCA_END_ENTITY_PROFILE_NAME")

	if pathToClientCert == "" || hostname == "" {
		t.Fatal("EJBCA_CLIENT_CERT_PATH and EJBCA_HOSTNAME must be set to run this test")
	}

	if ejbcaCaName == "" || ejbcaCertificateProfileName == "" || ejbcaEndEntityProfileName == "" {
		t.Fatal("EJBCA_CA_NAME, EJBCA_CERTIFICATE_PROFILE_NAME, and EJBCA_END_ENTITY_PROFILE_NAME must be set to run this test")
	}

	// Read the client cert and key from the file system.
	clientCertBytes, err := os.ReadFile(pathToClientCert)
	if err != nil {
		return
	}

	secretData := map[string][]byte{}
	secretData["hostname"] = []byte(hostname)
	secretData["clientCert.pem"] = clientCertBytes
	secretData["certificateProfileName"] = []byte(ejbcaCertificateProfileName)
	secretData["endEntityProfileName"] = []byte(ejbcaEndEntityProfileName)
	secretData["certificateAuthorityName"] = []byte(ejbcaCaName)

	// Create the signer
	signer, err := EjbcaSignerFromIssuerAndSecretData(nil, secretData)
	if err != nil {
		t.Fatal(err)
	}

	// Generate a CSR
	csr, err := generateCSR(os.Getenv("EJBCA_CSR_SUBJECT"))
	if err != nil {
		t.Fatal(err)
	}

	signedCert, err := signer.Sign(csr)
	if err != nil {
		return
	}

	t.Log(string(signedCert))
}

func generateCSR(subject string) ([]byte, error) {
	keyBytes, _ := rsa.GenerateKey(rand.Reader, 2048)

	subj, err := parseSubjectDN(subject, false)
	if err != nil {
		return make([]byte, 0, 0), err
	}

	template := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.SHA256WithRSA,
	}
	var csrBuf bytes.Buffer
	csrBytes, _ := x509.CreateCertificateRequest(rand.Reader, &template, keyBytes)
	err = pem.Encode(&csrBuf, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrBytes})
	if err != nil {
		return make([]byte, 0, 0), err
	}

	return csrBuf.Bytes(), nil
}

// Function that turns subject string into pkix.Name
// EG "C=US,ST=California,L=San Francisco,O=HashiCorp,OU=Engineering,CN=example.com"
func parseSubjectDN(subject string, randomizeCn bool) (pkix.Name, error) {
	var name pkix.Name

	// Split the subject into its individual parts
	parts := strings.Split(subject, ",")

	for _, part := range parts {
		// Split the part into key and value
		keyValue := strings.SplitN(part, "=", 2)

		if len(keyValue) != 2 {
			return pkix.Name{}, asn1.SyntaxError{Msg: "malformed subject DN"}
		}

		key := strings.TrimSpace(keyValue[0])
		value := strings.TrimSpace(keyValue[1])

		// Map the key to the appropriate field in the pkix.Name struct
		switch key {
		case "C":
			name.Country = []string{value}
		case "ST":
			name.Province = []string{value}
		case "L":
			name.Locality = []string{value}
		case "O":
			name.Organization = []string{value}
		case "OU":
			name.OrganizationalUnit = []string{value}
		case "CN":
			if randomizeCn {
				value = fmt.Sprintf("%s-%s", value, generateRandomString(5))
			} else {
				name.CommonName = value
			}
		default:
			// Ignore any unknown keys
		}
	}

	return name, nil
}
