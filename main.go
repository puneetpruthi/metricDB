package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/donnpebe/go-redis-timeseries"
	"github.com/garyburd/redigo/redis"
	"github.com/golang/protobuf/ptypes"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	pb "server/metric"
)

type MetricServiceServer struct {
	DB *timeseries.TimeSeries
}

const (
	GRPC_PORT          = ":6000"
	STATUS_OK          = 1
	DB_ADDR            = "redis:6379"
	DB_NAME            = "dump::metridDB"
	METRIC_GRANULARITY = 15
)

func (s *MetricServiceServer) GetMetric(ctx context.Context, getData *pb.GetRequest) (*pb.MetricData, error) {
	metricPlot, err := queryDB(s.DB, getData.GetUid(), getData)
	if err != nil {
		return &pb.MetricData{}, errors.New("db error:" + err.Error())
	}
	return &pb.MetricData{getData.GetUid(), metricPlot}, nil
}

func (s *MetricServiceServer) SetMetric(stream pb.MetricService_SetMetricServer) error {
	for {
		setReq, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return stream.SendAndClose(&pb.Status{
					RetStatus: STATUS_OK,
				})
			}
			return err
		}

		err = saveDB(s.DB, setReq.GetUid(), setReq.GetCount())
		if err != nil {
			return errors.New("db error:" + err.Error())
		}
	}
}

func queryDB(dbClient *timeseries.TimeSeries, uid string, getData *pb.GetRequest) ([]int64, error) {
	totalInterval := getData.GetInterval()
	fromTime, err := ptypes.Timestamp(getData.GetFromTime())
	if err != nil {
		return []int64{}, errors.New("invalid FROM time format")
	}
	toTime, err := ptypes.Timestamp(getData.GetToTime())
	if err != nil {
		return []int64{}, errors.New("invalid TO time format")
	}

	// verify request
	if fromTime.Unix() > toTime.Unix() ||
		totalInterval <= 0 ||
		(toTime.Unix()-fromTime.Unix())/totalInterval < METRIC_GRANULARITY {
		return []int64{}, errors.New("invalid request")
	}

	// calculate each slot duration to collate results in seconds
	//  = (End Time - Start Time) / No of Interval
	slotSeconds := (toTime.Unix() - fromTime.Unix()) / totalInterval
	slotDuration := time.Duration(slotSeconds) * time.Second

	var metricData []int64
	for begin := fromTime; begin.Unix() < toTime.Unix(); begin = begin.Add(slotDuration) {
		end := begin.Add(slotDuration)
		var uidCountPairs []string
		if err := dbClient.FetchRange(begin, end, &uidCountPairs); err != nil {
			return []int64{}, errors.New("db error:" + err.Error())
		}

		// uidCountPairs[] has entries of the form
		//  <uid:count> or <String:Integer>
		//
		// we need to extract the count for each uid in time interval here
		// we now need to:
		//   - pick count only for uid requested
		//   - sum and store count for N equal intervals queried for
		var currentIntervalCount int64
		currentIntervalCount = 0
		for _, uidCount := range uidCountPairs {
			splitter := strings.Split(uidCount, ":")
			if len(splitter) != 3 {
				return []int64{}, errors.New("db error: invalid entry found in database")
			}
			if strings.Compare(splitter[0], uid) != 0 {
				continue
			}
			thisCount, err := strconv.ParseInt(splitter[1], 10, 64)
			if err != nil {
				return []int64{}, errors.New("db error: invalid entry found in database")
			}
			currentIntervalCount += thisCount
		}
		metricData = append(metricData, currentIntervalCount)
	}

	// return the metricData array
	return metricData, nil
}

// save the data in an encoded string for easy decoding in
// time series database. Each entry is stored as:
// <uid>:<count>:<timestamp-nanosecond>
// in the database such that if any other client stores the same
// count, it is stored as an individual entry
func saveDB(dbClient *timeseries.TimeSeries, uid string, count int64) error {
	saveTime := time.Now()
	saveData := fmt.Sprintf("%s:%d:%d", uid, count, saveTime.UnixNano())
	return dbClient.Add(saveData, saveTime)
}

// connect to time series database
func bootstrapDatabase() (*timeseries.TimeSeries, error) {
	dbConn, err := redis.Dial("tcp", DB_ADDR)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	timeseriesDB := timeseries.NewTimeSeries(DB_NAME, METRIC_GRANULARITY*time.Second, 0, dbConn)
	return timeseriesDB, nil
}

func main() {
	// connect to database
	dbClient, err := bootstrapDatabase()
	if err != nil {
		panic("could not connect to database")
	}

	// initiate the server
	grpcServer := grpc.NewServer()
	pb.RegisterMetricServiceServer(grpcServer, &MetricServiceServer{DB: dbClient})

	// listen on GRPC port for incoming requests
	listen, err := net.Listen("tcp", GRPC_PORT)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Println("Listening on tcp://localhost:" + GRPC_PORT)
	grpcServer.Serve(listen)
}
