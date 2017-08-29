package client

import (
	"log"
	"time"

	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"

	pb "client/metric"
)

// printFeature gets the feature for the given point.
func getCount(client pb.MetricServiceClient, uid string) {
	now := time.Now()
	fromTime, _ := ptypes.TimestampProto(now.Add(-150 * time.Second))
	toTime, _ := ptypes.TimestampProto(now)

	getRequest := &pb.GetRequest{
		Uid:      uid,
		FromTime: fromTime,
		ToTime:   toTime,
		Interval: 10,
	}

	data, err := client.GetMetric(context.Background(), getRequest)
	if err != nil {
		log.Fatalf("%v.GetFeatures(_) = _, %v: ", client, err)
	}
	log.Printf("GetCount for uid: %s :%#v", uid, data.GetCounts())
}

func setCount(client pb.MetricServiceClient, uid string, Count int64, times int) {
	stream, err := client.SetMetric(context.Background())
	if err != nil {
		return
	}

	msg := &pb.SetRequest{
		Uid:   uid,
		Count: Count,
	}
	for i := 0; i < times; i++ {
		if err := stream.Send(msg); err != nil {
			log.Fatalf("%v.Send(%v) = %v", stream, msg, err)
		}
		time.Sleep(1 * time.Second)
	}

	reply, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatalf("%v.CloseAndRecv() got error %v, want %v", stream, err, nil)
	}
	log.Printf("Route summary: %v", reply)
}

func main() {
	conn, err := grpc.Dial("localhost:6000", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %s", err)
	}
	defer conn.Close()

	client := pb.NewMetricServiceClient(conn)

	setCount(client, "8ed5", 3, 10)
	setCount(client, "abcd", 6, 5)
	getCount(client, "8ed5")
	getCount(client, "abcd")
}
