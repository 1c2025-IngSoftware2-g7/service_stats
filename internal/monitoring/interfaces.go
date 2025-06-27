package monitoring

import "github.com/newrelic/go-agent/v3/newrelic"

type NewRelicApp struct {
	App *newrelic.Application
}

func (n *NewRelicApp) StartTransaction(name string) Transaction {
	txn := n.App.StartTransaction(name, nil, nil)
	return &NewRelicTransaction{txn}
}

type NewRelicTransaction struct {
	txn *newrelic.Transaction
}

func (t *NewRelicTransaction) End() {
	t.txn.End()
}
