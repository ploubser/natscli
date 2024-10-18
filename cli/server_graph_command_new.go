package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/choria-io/fisk"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/natscli/internal/tui"
	iu "github.com/nats-io/natscli/internal/util"
)

type SrvGraphCmd struct {
	id string
	js bool
}

func configureServerGraphCommand(srv *fisk.CmdClause) {
	c := &SrvGraphCmd{}

	graph := srv.Command("graph", "Show graphs for a single server").Action(c.graph)
	graph.Arg("server", "Server ID or Name to inspect").StringVar(&c.id)
	graph.Flag("jetstream", "Draw JetStream statistics").Short('j').UnNegatableBoolVar(&c.js)
}

func (c *SrvGraphCmd) graph(_ *fisk.ParseContext) error {
	if !c.js {
		return c.graphServer()
	}
	return nil
}

func (c SrvGraphCmd) graphServer() error {
	if !iu.IsTerminal() {
		return fmt.Errorf("can only graph data on an interactive terminal")
	}

	nc, _, err := prepareHelper("", natsOpts()...)
	if err != nil {
		return err
	}

	defer nc.Close()

	m, err := tui.NewGraphApp(c.id)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	header := tui.NewHeaderWindow(-1, 3)
	m.SetHeader(header)

	help := tui.NewHelpWindow(-1, 3, "(q) - Quit Application")
	m.SetHelpWindow(help)

	cpu := m.AddChart("CPU %% Used (normalized for %.d cores) : %.2f%%", 0) // change to update range
	mem := m.AddChart("Memory Used in MB : %.2fMB", 100)
	conn := m.AddChart("Connections : %0.f", 100)
	subs := m.AddChart("Subscriptions : %0.f", 200)
	mio := m.AddChart("Messages In+Out / second : %0.f", 1000)
	bio := m.AddChart("Bytes In+Out / second : %0.f bytes", 1000)

	m.SetTickerCallback(func(active string) {
		m.LastUpdate = time.Now()
		vz, err := getVz(nc, active)
		if err != nil {
			// Log something here
			os.Exit(1)
		}
		cpuav := vz.CPU / float64(vz.Cores)
		cpu.Push(cpuav)
		cpu.UpdateTitle(vz.Cores, cpuav)

		memav := float64(vz.Mem / 1024 / 1024)
		mem.Push(memav)
		mem.UpdateTitle(memav)

		conn.Push(float64(vz.Connections))
		conn.UpdateTitle(float64(vz.Connections))

		subs.Push(float64(vz.Subscriptions))
		subs.UpdateTitle(float64(vz.Subscriptions))

		if mio.LastValue == 0 {
			mio.LastValue = float64(vz.InMsgs + vz.OutMsgs)
		}
		mio.Push((float64(vz.InMsgs+vz.OutMsgs) - mio.LastValue) / time.Since(m.LastUpdate).Seconds())
		mio.UpdateTitle(mio.LastValue)

		if bio.LastValue == 0 {
			bio.LastValue = float64(vz.InBytes + vz.OutBytes)
		}
		bio.Push((float64(vz.InBytes+vz.OutBytes) - bio.LastValue) / time.Since(m.LastUpdate).Seconds())
		bio.UpdateTitle(bio.LastValue)
	})

	servers, err := getServers(nc)
	if err != nil {
		log.Printf("Could not retreive server list: %s", err)
		os.Exit(1)
	}

	m.CreateMenu(servers)

	if _, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion()).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}

	return nil
}

func getServers(nc *nats.Conn) ([]string, error) {
	servers := []string{}

	var err error
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*500)

	mu := &sync.Mutex{}
	start := time.Now()
	times := []float64{}

	sub, err := nc.Subscribe(nc.NewRespInbox(), func(msg *nats.Msg) {
		if msg.Header != nil && msg.Header.Get("Status") != "" {
			fmt.Printf("%s status from $SYS.REQ.SERVER.PING, ensure a system account is used with appropriate permissions\n", msg.Header.Get("Status"))
			os.Exit(1)
		}

		ssm := &server.ServerStatsMsg{}
		err = json.Unmarshal(msg.Data, ssm)
		if err != nil {
			log.Printf("Could not decode response: %s", err)
			os.Exit(1)
		}
		servers = append(servers, ssm.Server.Name)

		mu.Lock()
		defer mu.Unlock()

		since := time.Since(start)
		rtt := since.Milliseconds()
		times = append(times, float64(rtt))

	})
	if err != nil {
		log.Printf("Error processing response: %s", err)
		os.Exit(1)
	}

	err = nc.PublishRequest("$SYS.REQ.SERVER.PING", sub.Subject, nil)
	if err != nil {
		log.Printf("Error publishing request: %s", err)
		os.Exit(1)
	}

	ic := make(chan os.Signal, 1)
	signal.Notify(ic, os.Interrupt)

	select {
	case <-ic:
		cancel()
	case <-ctx.Done():
	}

	err = sub.Drain()
	if err != nil {
		log.Printf("Failed to drain all responses: %s", err)
		os.Exit(1)
	}

	return servers, nil
}

func getVz(nc *nats.Conn, id string) (*server.Varz, error) {
	varz := &server.Varz{}
	subj := fmt.Sprintf("$SYS.REQ.SERVER.%s.VARZ", id)
	body := []byte("{}")
	var err error

	if len(id) != 56 || strings.ToUpper(id) != id {
		subj = "$SYS.REQ.SERVER.PING.VARZ"
		opts := server.VarzEventOptions{EventFilterOptions: server.EventFilterOptions{Name: id}}
		body, err = json.Marshal(opts)
		if err != nil {
			return varz, err
		}
	}

	resp, err := nc.Request(subj, body, opts().Timeout)
	if err != nil {
		return nil, fmt.Errorf("no results received, ensure the account used has system privileges and appropriate permissions")
	}

	reqresp := map[string]json.RawMessage{}
	err = json.Unmarshal(resp.Data, &reqresp)
	if err != nil {
		return nil, err
	}

	errresp, ok := reqresp["error"]
	if ok {
		return nil, fmt.Errorf("invalid response received: %#v", errresp)
	}

	data, ok := reqresp["data"]
	if !ok {
		return nil, fmt.Errorf("no data received in response: %#v", reqresp)
	}

	err = json.Unmarshal(data, varz)
	if err != nil {
		return nil, err
	}

	return varz, nil
}
