package certinator

import (
	"fmt"
	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

func (c *Certinator) CaExists(name string) (ok bool, err error) {
	path := "sys/mounts"
	secret, err := c.Client.Logical().Read(path)
	if err != nil {
		err = errors.Wrapf(err, "failed to read %s", path)
	}

	mountName := fmt.Sprintf("%s/", name)

	for k, v := range secret.Data {
		if k == mountName {
			d, ok := v.(map[string]interface{})
			if ok {
				if d["type"] == "pki" {
					ok = true
					return ok, err
				}
			}
		}
	}
	return ok, err
}

// CreateCa  Creates a CA.  Equivalent of running 'vault secrets enable -path=<name> -description="<description" -max-lease-ttl=43800h pki'
func (c *Certinator) CreateCa(name string) (err error) {
	config := map[string]interface{}{
		"options":           nil,
		"default_lease_ttl": "0s",
		"max_lease_ttl":     DEFAULT_CA_MAX_LEASE,
		"force_no_cache":    false,
	}

	data := map[string]interface{}{
		"type":        "pki",
		"description": fmt.Sprintf("%s certificate authority", name),
		"config":      config,
	}

	path := fmt.Sprintf("sys/mounts/%s", name)

	_, err = c.Client.Logical().Write(path, data)
	if err != nil {
		err = errors.Wrapf(err, "failed creating %s CA", name)
		return err
	}

	return err
}

// TuneCA Tunes the CA.  Equivalent of running 'vault secrets tune -max-lease-ttl=43800h <name>'
func (c *Certinator) TuneCA(name string) (err error) {
	data := map[string]interface{}{
		"options":           nil,
		"default_lease_ttl": "",
		"max_lease_ttl":     DEFAULT_CA_MAX_LEASE,
		"force_no_cache":    false,
	}

	path := fmt.Sprintf("sys/mounts/%s/tune", name)

	_, err = c.Client.Logical().Write(path, data)
	if err != nil {
		err = errors.Wrapf(err, "failed creating %s CA", name)
		return err
	}

	return err
}

// DeleteCA Deletes a CA.  Equivalent of running 'vault secrets disable <name>'
func (c *Certinator) DeleteCA(name string) (err error) {
	path := fmt.Sprintf("sys/mounts/%s", name)

	_, err = c.Client.Logical().Delete(path)
	if err != nil {
		err = errors.Wrapf(err, "Failed to delete CA at %s", name)
		return err
	}

	return err
}

// Generate CA Cert generates the CA cert.  Equivalent to running 'vault write <name>/root/generate/internal common_name=<common name> ttl=43800h' or 'vault write <name>/root/generate/exported common_name=<common name> ttl=43800h' returns the secret generated, which may or may not contain the CA Private Key, depending on how you called the function.
func (c *Certinator) GenerateCaCert(name string, cn string, exported bool) (secret *api.Secret, err error) {
	data := map[string]interface{}{
		"ttl":         DEFAULT_CA_MAX_LEASE,
		"common_name": cn,
	}

	var path string
	if exported {
		path = fmt.Sprintf("%s/root/generate/exported", name)
	} else {
		path = fmt.Sprintf("%s/root/generate/internal", name)
	}

	secret, err = c.Client.Logical().Write(path, data)
	if err != nil {
		err = errors.Wrapf(err, "failed creating root cert for CA %q", name)
		return secret, err
	}

	return secret, err
}

//ConfigureCRL Configure the CRL endpoint.  Eqivalent to running 'vault write <name>/config/urls issuing_certificates=<vaultUrl>/v1/<name>/ca crl_distribution_points=<vaultUrl>/v1/<name>/crl
func (c *Certinator) ConfigureCRL(name string, vaultUrl string) (err error) {
	data := map[string]interface{}{
		"issuing_certificates":    fmt.Sprintf("%s/v1/%s/ca", vaultUrl, name),
		"crl_distribution_points": fmt.Sprintf("%s/v1/%s/crl", vaultUrl, name),
	}

	path := fmt.Sprintf("%s/config/urls", name)

	_, err = c.Client.Logical().Write(path, data)
	if err != nil {
		err = errors.Wrapf(err, "failed configuring URLs for CA %q", name)
		return err
	}

	return err
}

// CreateIssuingRole Create a role with which to issue certificates.  Equivalent to running 'vault write <caName>/roles/<roleName> allowed_domains="<domain>" allow_subdomains="true" max_ttl="8760h" ttl="8760h" allow_ip_sans=true allow_localhost=true
func (c *Certinator) CreateIssuingRole(caName string, role CertificateIssuingRole) (err error) {

	data := map[string]interface{}{
		"allowed_domains":  role.Domains,
		"allow_subdomains": role.Subdomains,
		"ip_sans":          role.IpSans,
		"allow_localhost":  role.Localhost,
		"max_ttl":          role.MaxTTL,
		"ttl":              role.Ttl,
	}

	path := fmt.Sprintf("%s/roles/%s", caName, role.Name)

	_, err = c.Client.Logical().Write(path, data)
	if err != nil {
		err = errors.Wrapf(err, "failed creating role %s for CA %q", role.Name, caName)
		return err
	}

	return err
}