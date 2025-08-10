package verifiers

import (
	"fmt"
	"net"
	"slices"

	"github.com/geekgonecrazy/vanityDomainManager/jobs"
)

// VerifyDomain checks the DNS records of a vanity domain against the desired targets.
func VerifyDomain(domain jobs.VanityDomain) error {
	switch domain.DesiredDNSTargetType {
	case "CNAME":
		cname, err := net.LookupCNAME(domain.VanityDomain)
		if err != nil || cname == "" {
			return fmt.Errorf("Error or empty cname: %v", err)
		}

		if len(cname) > 1 && cname[:len(cname)-1] != domain.DesiredCNAMETarget {
			return fmt.Errorf("Incorrect CNAME value: %s, expected: %s", cname, domain.DesiredCNAMETarget)
		}
	case "A":
		ips, err := net.LookupIP(domain.VanityDomain)
		if err != nil || len(ips) == 0 {
			return fmt.Errorf("Error or empty A record: %v", err)
		}

		// check if all ips match the desired targets
		for _, ip := range ips {
			found := slices.Contains(domain.DesiredARecordTargets, ip.String())
			if !found {
				return fmt.Errorf("Incorrect A record value: %s, expected one of: %v", ip.String(), domain.DesiredARecordTargets)
			}
		}
	default:
		return fmt.Errorf("Unsupported DNS target type: %s", domain.DesiredDNSTargetType)
	}

	return nil
}
