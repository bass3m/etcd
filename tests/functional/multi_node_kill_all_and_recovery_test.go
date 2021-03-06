package test

import (
	"bytes"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/coreos/etcd/server"
	"github.com/coreos/etcd/tests"
	"github.com/coreos/etcd/third_party/github.com/coreos/go-etcd/etcd"
	"github.com/coreos/etcd/third_party/github.com/stretchr/testify/assert"
)

// Create a five nodes
// Kill all the nodes and restart
func TestMultiNodeKillAllAndRecovery(t *testing.T) {
	procAttr := new(os.ProcAttr)
	procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}

	stop := make(chan bool)
	leaderChan := make(chan string, 1)
	all := make(chan bool, 1)

	clusterSize := 5
	argGroup, etcds, err := CreateCluster(clusterSize, procAttr, false)
	defer DestroyCluster(etcds)

	if err != nil {
		t.Fatal("cannot create cluster")
	}

	c := etcd.NewClient(nil)

	go Monitor(clusterSize, clusterSize, leaderChan, all, stop)
	<-all
	<-leaderChan
	stop <- true

	c.SyncCluster()

	// send 10 commands
	for i := 0; i < 10; i++ {
		// Test Set
		_, err := c.Set("foo", "bar", 0)
		if err != nil {
			panic(err)
		}
	}

	time.Sleep(time.Second)

	// kill all
	DestroyCluster(etcds)

	time.Sleep(time.Second)

	stop = make(chan bool)
	leaderChan = make(chan string, 1)
	all = make(chan bool, 1)

	time.Sleep(time.Second)

	for i := 0; i < clusterSize; i++ {
		etcds[i], err = os.StartProcess(EtcdBinPath, argGroup[i], procAttr)
	}

	go Monitor(clusterSize, 1, leaderChan, all, stop)

	<-all
	<-leaderChan

	result, err := c.Set("foo", "bar", 0)

	if err != nil {
		t.Fatalf("Recovery error: %s", err)
	}

	if result.Node.ModifiedIndex != 17 {
		t.Fatalf("recovery failed! [%d/17]", result.Node.ModifiedIndex)
	}
}

// TestTLSMultiNodeKillAllAndRecovery create a five nodes
// then kill all the nodes and restart
func TestTLSMultiNodeKillAllAndRecovery(t *testing.T) {
	procAttr := new(os.ProcAttr)
	procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}

	stop := make(chan bool)
	leaderChan := make(chan string, 1)
	all := make(chan bool, 1)

	clusterSize := 5
	argGroup, etcds, err := CreateCluster(clusterSize, procAttr, true)
	defer DestroyCluster(etcds)

	if err != nil {
		t.Fatal("cannot create cluster")
	}

	c := etcd.NewClient(nil)

	go Monitor(clusterSize, clusterSize, leaderChan, all, stop)
	<-all
	<-leaderChan
	stop <- true

	c.SyncCluster()

	// send 10 commands
	for i := 0; i < 10; i++ {
		// Test Set
		_, err := c.Set("foo", "bar", 0)
		if err != nil {
			panic(err)
		}
	}

	time.Sleep(time.Second)

	// kill all
	DestroyCluster(etcds)

	time.Sleep(time.Second)

	stop = make(chan bool)
	leaderChan = make(chan string, 1)
	all = make(chan bool, 1)

	time.Sleep(time.Second)

	for i := 0; i < clusterSize; i++ {
		etcds[i], err = os.StartProcess(EtcdBinPath, argGroup[i], procAttr)
		// See util.go for the reason to wait for server
		client := buildClient()
		err = WaitForServer("127.0.0.1:400"+strconv.Itoa(i+1), client, "http")
		if err != nil {
			t.Fatalf("node start error: %s", err)
		}
	}

	go Monitor(clusterSize, 1, leaderChan, all, stop)

	<-all
	<-leaderChan

	result, err := c.Set("foo", "bar", 0)

	if err != nil {
		t.Fatalf("Recovery error: %s", err)
	}

	if result.Node.ModifiedIndex != 17 {
		t.Fatalf("recovery failed! [%d/17]", result.Node.ModifiedIndex)
	}
}

// Create a five-node cluster
// Kill all the nodes and restart
func TestMultiNodeKillAllAndRecoveryWithStandbys(t *testing.T) {
	procAttr := new(os.ProcAttr)
	procAttr.Files = []*os.File{nil, os.Stdout, os.Stderr}

	stop := make(chan bool)
	leaderChan := make(chan string, 1)
	all := make(chan bool, 1)

	clusterSize := 5
	argGroup, etcds, err := CreateCluster(clusterSize, procAttr, false)
	defer DestroyCluster(etcds)

	if err != nil {
		t.Fatal("cannot create cluster")
	}

	c := etcd.NewClient(nil)

	go Monitor(clusterSize, clusterSize, leaderChan, all, stop)
	<-all
	<-leaderChan
	stop <- true

	c.SyncCluster()

	// Reconfigure with smaller active size (3 nodes) and wait for demotion.
	resp, _ := tests.Put("http://localhost:7001/v2/admin/config", "application/json", bytes.NewBufferString(`{"activeSize":3}`))
	if !assert.Equal(t, resp.StatusCode, 200) {
		t.FailNow()
	}

	time.Sleep(2*server.ActiveMonitorTimeout + (1 * time.Second))

	// Verify that there is three machines in peer mode.
	result, err := c.Get("_etcd/machines", false, true)
	assert.NoError(t, err)
	assert.Equal(t, len(result.Node.Nodes), 3)

	// send 10 commands
	for i := 0; i < 10; i++ {
		// Test Set
		_, err := c.Set("foo", "bar", 0)
		if err != nil {
			panic(err)
		}
	}

	time.Sleep(time.Second)

	// kill all
	DestroyCluster(etcds)

	time.Sleep(time.Second)

	stop = make(chan bool)
	leaderChan = make(chan string, 1)
	all = make(chan bool, 1)

	time.Sleep(time.Second)

	for i := 0; i < clusterSize; i++ {
		etcds[i], err = os.StartProcess(EtcdBinPath, argGroup[i], procAttr)
	}

	time.Sleep(2 * time.Second)

	// send 10 commands
	for i := 0; i < 10; i++ {
		// Test Set
		_, err := c.Set("foo", "bar", 0)
		if err != nil {
			t.Fatalf("Recovery error: %s", err)
		}
	}

	// Verify that we have three machines.
	result, err = c.Get("_etcd/machines", false, true)
	assert.NoError(t, err)
	assert.Equal(t, len(result.Node.Nodes), 3)
}
