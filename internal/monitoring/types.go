package monitoring

type Application interface {
	StartTransaction(name string) Transaction
}

type Transaction interface {
	End()
}
