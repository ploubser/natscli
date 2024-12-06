// Copyright 2024 The NATS Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"github.com/choria-io/fisk"
	"github.com/nats-io/jsm.go"
	"github.com/nats-io/jsm.go/api"
	"github.com/nats-io/jsm.go/balancer"
	"github.com/nats-io/nats.go"
)

type BalanceCommand struct {
	query string
	nc    *nats.Conn
	mgr   *jsm.Manager
}

func configureBalanceCommand(app commandHost) {
	bc := &BalanceCommand{}
	balance := app.Command("balance-streams", "Balances streams across servers").Action(bc.Balance)
	balance.Flag("query", "Query that returns some streams").Required().StringVar(&bc.query)
}

func (b *BalanceCommand) Balance(_ *fisk.ParseContext) error {
	var err error
	opts := []jsm.StreamQueryOpt{}

	if b.query != "" {
		opts = append(opts, jsm.StreamQueryExpression(b.query))
	}

	b.nc, b.mgr, err = prepareHelper("", natsOpts()...)
	if err != nil {
		return nil
	}

	streams, err := b.mgr.QueryStreams(opts...)
	if err != nil {
		return err
	}

	if len(streams) > 0 {
		balancer, err := balancer.New(b.mgr.NatsConn(), api.NewDefaultLogger(api.InfoLevel))
		if err != nil {
			return err
		}

		balancer.BalanceStreams(streams)
	}
	return nil
}
