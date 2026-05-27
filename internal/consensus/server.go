package consensus

import (
    "context"
    "io"
    "net"

    pb "github.com/trusttrace/trusttrace/proto/metrics"
    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// GRPCServer is the consensus ingestion gRPC server.
type GRPCServer struct {
    pb.UnimplementedMetricsIngestionServer
    quorum *QuorumManager
    dedup  *Deduplicator
    log    *zap.Logger
}

// NewGRPCServer constructs the server with injected quorum and dedup logic.
func NewGRPCServer(qm *QuorumManager, dd *Deduplicator, log *zap.Logger) *GRPCServer {
    return &GRPCServer{quorum: qm, dedup: dd, log: log}
}

// Ingest handles a single unary probe ingest RPC.
func (s *GRPCServer) Ingest(_ context.Context, req *pb.IngestRequest) (*pb.IngestResponse, error) {
    r := req.Result
    if r == nil {
        return nil, status.Error(codes.InvalidArgument, "nil result")
    }
    if s.dedup.SeenOrAdd(r.ProbeId) {
        return &pb.IngestResponse{Accepted: false, Reason: "duplicate probe_id"}, nil
    }
    s.quorum.Add(r)
    return &pb.IngestResponse{Accepted: true}, nil
}

// IngestStream handles a bidirectional stream of probes.
func (s *GRPCServer) IngestStream(stream pb.MetricsIngestion_IngestStreamServer) error {
    for {
        req, err := stream.Recv()
        if err == io.EOF {
            return nil
        }
        if err != nil {
            return status.Errorf(codes.Internal, "recv: %v", err)
        }
        r := req.Result
        if r == nil {
            continue
        }
        if s.dedup.SeenOrAdd(r.ProbeId) {
            if err := stream.Send(&pb.IngestResponse{Accepted: false, Reason: "duplicate"}); err != nil {
                return err
            }
            continue
        }
        s.quorum.Add(r)
        if err := stream.Send(&pb.IngestResponse{Accepted: true}); err != nil {
            return err
        }
    }
}

// ListenAndServe starts the gRPC server on addr.
func (s *GRPCServer) ListenAndServe(addr string) error {
    lis, err := net.Listen("tcp", addr)
    if err != nil {
        return err
    }
    srv := grpc.NewServer(
        grpc.ChainUnaryInterceptor(loggingInterceptor(s.log)),
    )
    pb.RegisterMetricsIngestionServer(srv, s)
    s.log.Info("consensus gRPC listening", zap.String("addr", addr))
    return srv.Serve(lis)
}

func loggingInterceptor(log *zap.Logger) grpc.UnaryServerInterceptor {
    return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
        resp, err := handler(ctx, req)
        if err != nil {
            log.Warn("rpc error", zap.String("method", info.FullMethod), zap.Error(err))
        }
        return resp, err
    }
}
