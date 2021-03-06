package main

import (
	"encoding/json"
	"flag"

	"github.com/golang/glog"
	elastic "gopkg.in/olivere/elastic.v3"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Client is kubernetes.Client
type Client struct {
	Config *rest.Config
	Conn   *kubernetes.Clientset
}

// Init is kuberenets.Init
func (f *Client) Init(kubeconfig string) {
	if kubeconfig != "" {
		config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			glog.Fatal("Kubeconfig File Error: ", err.Error())
		}
		f.Config = config
	} else {
		config, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatal("Kubeconfig In Cluster Error: ", err.Error())
		}
		f.Config = config
	}

	client, err := kubernetes.NewForConfig(f.Config)
	if err != nil {
		glog.Fatal("Kubernetes Client Error: ", err.Error())
	}
	f.Conn = client
}

// GetEvents is kubernetes.GetEvents
// https://github.com/kubernetes/client-go/issues/152
func (f *Client) GetEvents() (watchInterface watch.Interface, err error) {
	watchInterface, err = f.Conn.CoreV1().Events("").
		Watch(meta_v1.ListOptions{
			Watch: true,
		})
	return watchInterface, err
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Path to a kubeconfig file")
	elasticsearchHost := flag.String("elasticsearchHost", "http://127.0.0.1:9200/", "Elasticsearch Host Url")
	elasticsearchIndex := flag.String("elasticsearchIndex", "kubernetesevents", "Elasticsearch Index Name")
	elasticsearchType := flag.String("elasticsearchType", "kubernetestable", "Elasticsearch Type Name")

	flag.Parse()

	glog.Info("Kubernetes Conn Init")
	kc := Client{}
	kc.Init(*kubeconfig)

	glog.Info("Kubernetes Conn Get Events")
	eventWatch, err := kc.GetEvents()
	if err != nil {
		glog.Fatal(err.Error())
	}

	glog.Info("Kubernetes Conn Get Events Chan")

	glog.Info("Conn Elasticsearch")
	connes, err := elastic.NewClient(elastic.SetURL(*elasticsearchHost))
	if err != nil {
		glog.Fatal(err.Error())
	}

	eventWatchChan := eventWatch.ResultChan()
	for {
		select {
		case event := <-eventWatchChan:
			eventJSON, err := json.Marshal(event)
			if err != nil {
				glog.Error(err.Error())
				continue
			}

			_, err = connes.Index().Index(*elasticsearchIndex).Type(*elasticsearchType).BodyJson(eventJSON).Do()
			if err != nil {
				glog.Fatal(err.Error())
			} else {
				glog.Info(eventJSON)
			}
		}
	}

}
