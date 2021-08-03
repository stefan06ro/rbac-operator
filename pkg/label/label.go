package label

const (
	Organization = "giantswarm.io/organization"

	ManagedBy = "giantswarm.io/managed-by"

	// Labels, used in legacy cluster namespaces
	LegacyCustomer = "customer"
)

const (
	NotManagedByHelm = "app.kubernetes.io/managed-by!=Helm"
)
