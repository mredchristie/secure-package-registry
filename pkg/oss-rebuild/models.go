package ossrebuild

type Ecosystem string

const (
	EcosystemNPM    Ecosystem = "npm"
	EcosystemPyPI   Ecosystem = "pypi"
	EcosystemCrates Ecosystem = "crates"
)

type Attestation struct {
	Ecosystem         Ecosystem
	Package           string
	Version           string
	HasAttestation    bool
	AttestationBundle []byte
}
