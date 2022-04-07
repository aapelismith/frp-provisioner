package ingress_test

//import (
//	"context"
//	"k8s.io/client-go/informers"
//	"kunstack.com/pharos/pkg/client/clientset"
//	"kunstack.com/pharos/pkg/controller/ingress"
//	"kunstack.com/pharos/pkg/log"
//	"kunstack.com/pharos/pkg/safe"
//	"sync"
//	"testing"
//	"time"
//)
//
//func Test_Controller(t *testing.T) {
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
//	defer cancel()
//
//	log.SetLevel(log.TraceLevel)
//	log.SetCallerEncoder(log.ShortCaller())
//
//	// 创建 client
//	client, err := clientset.NewClient(clientset.NewOptions())
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	informer := informers.NewSharedInformerFactory(client, time.Second*600)
//
//	ctl, err := ingress.NewController(context.Background(), ingress.NewOptions(), informer.Core().V1().Services(), client)
//
//	if err != nil {
//		t.Fatal(err)
//	}
//
//	wg := sync.WaitGroup{}
//
//	wg.Add(2)
//
//	safe.Go(func() {
//		defer wg.Done()
//		ctl.Run(ctx, 6)
//	})
//
//	safe.Go(func() {
//		defer wg.Done()
//		informer.Start(ctx.Done())
//	})
//
//	<-ctx.Done()
//	wg.Wait()
//}
