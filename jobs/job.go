package jobs

type DomainCustomCert struct {
	Key  string `json:"key"`
	Cert string `json:"cert"`
}

type VanityDomain struct {
	VanityDomain          string            `json:"vanityDomain"`
	DesiredDNSTargetType  string            `json:"desiredDnsTargetType"` // CNAME or A
	DesiredCNAMETarget    string            `json:"desiredCNAME"`
	DesiredARecordTargets []string          `json:"desiredARecords"`
	ProvidedCertificate   *DomainCustomCert `json:"providedCertificate,omitempty"` // Optional, if the user provides a certificate
	TargetServiceName     string            `json:"targetServiceName,omitempty"`   // Optional, The service name to set in the ingress
	TargetServicePort     int32             `json:"targetServicePort,omitempty"`   // Optional, The service port to set in the ingress
}

type DomainJob struct {
	Type        string       `json:"type"`        // "add", "change", or "remove"
	Domain      VanityDomain `json:"domain"`      // The vanity domain to process
	ReferenceID string       `json:"referenceId"` // Unique ID for the job, can be used to track the job
}

type JobStatus struct {
	Success      bool   `json:"success"`      // Whether the job was successful
	ReferenceID  string `json:"referenceId"`  // Unique ID for the job, same as in DomainJob
	ErrorMessage string `json:"errorMessage"` // Error message if the job failed
	Dropped      bool   `json:"dropped"`      // Whether the job was dropped
}
