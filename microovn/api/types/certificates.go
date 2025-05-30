// Package types provides shared types and structs.
package types

import "fmt"

// IssueCertificateResponse is a structure that models response to requests for issuance
// of OVN certificates.
type IssueCertificateResponse struct {
	Success []string `json:"success" yaml:"success"` // List of services for which new certificates were issued
	Failed  []string `json:"failed" yaml:"failed"`   // List of services for which new certificates issue failed
}

// PrettyPrint method formats and prints contents of IssueCertificateResponse object
func (i *IssueCertificateResponse) PrettyPrint() {
	fmt.Println("Local certificates reissued successfully:")
	if i.Success == nil {
		fmt.Println("None")
	} else {
		for _, service := range i.Success {
			fmt.Println(service)
		}
	}

	if i.Failed != nil {
		fmt.Println("\nLocal certificates reissue failed:")
		for _, service := range i.Failed {
			fmt.Println(service)
		}
	}
}

// RegenerateCaResponse is a structure that models response to requests for complete CA rebuild
type RegenerateCaResponse struct {
	NewCa                bool                                `json:"newCa" yaml:"newCa"`                               // True if this node issued new CA cert
	ReissuedCertificates map[string]IssueCertificateResponse `json:"reissuedCertificates" yaml:"reissuedCertificates"` // map of hosts and service certificates they issued
	Errors               []string                            `json:"error"`                                            //
}

// PrettyPrint method formats and prints contents of RegenerateCaResponse object
func (r *RegenerateCaResponse) PrettyPrint() {
	var newCaSuccess string
	if r.NewCa {
		newCaSuccess = "Issued"
	} else {
		newCaSuccess = "Not Issued"
	}

	fmt.Printf("New CA certificate: %s\n\n", newCaSuccess)
	fmt.Print("Service certificate re-issued for following services:")

	anyFailure := false
	for host, certificates := range r.ReissuedCertificates {
		fmt.Printf("\n[Host %s]\n", host)
		for _, service := range certificates.Success {
			fmt.Printf("%s: Success\n", service)
		}
		for _, service := range certificates.Failed {
			fmt.Printf("%s: Failed!\n", service)
			anyFailure = true
		}
	}

	if len(r.Errors) != 0 {
		anyFailure = true
		fmt.Println("\n[Errors]")
		for _, errMsg := range r.Errors {
			fmt.Println(errMsg)
		}
	}

	if anyFailure {
		fmt.Println(
			"\n Some of the service certificates failed to be re-issued. You can inspect logs and " +
				"attempt to re-issue these certificates using microovn CLI on affected cluster members.",
		)
	}

}

// NewRegenerateCaResponse returns pointer to initialized RegenerateCaResponse object
func NewRegenerateCaResponse() *RegenerateCaResponse {
	return &RegenerateCaResponse{
		NewCa:                false,
		ReissuedCertificates: make(map[string]IssueCertificateResponse),
		Errors:               make([]string, 0),
	}
}

// CaInfo is a response to GET /1.0/ca and returns additional information about
// the CA certificate.
type CaInfo struct {
	AutoRenew bool   `json:"auto_renew"`
	Error     string `json:"error"`
}
