package reloadconfig

import "fmt"

// getServiceCode returns the service code = <service name>-<chain ID>
func getServiceCode(serviceName string, chainID int) string {
	return fmt.Sprintf("%s-%d", serviceName, chainID)
}
