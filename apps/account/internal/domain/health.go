package domain

// HealthStatus represents the health of the service dependencies.
type HealthStatus struct {
	Status         string
	DBConnected    bool
	RedisConnected bool
}
