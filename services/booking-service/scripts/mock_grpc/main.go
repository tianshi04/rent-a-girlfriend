package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"

	financev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/financev1"
	profilev1 "github.com/rent-a-girlfriend/booking-service/gen/proto/profilev1"
)

// -----------------------------------------------------------------------------
// PROFILE SERVICE MOCK
// -----------------------------------------------------------------------------
type mockProfileServer struct {
	profilev1.UnimplementedProfileServiceServer
}

func (s *mockProfileServer) GetScenarioSnapshot(ctx context.Context, req *profilev1.GetScenarioSnapshotRequest) (*profilev1.ScenarioSnapshotResponse, error) {
	log.Printf("📥 [gRPC Profile] Received GetScenarioSnapshot for ScenarioID: %s", req.ScenarioId)

	// Default Mock Data
	price := int64(600)
	duration := int32(90)

	// Dynamic Mock Behavior for testing specific scenarios
	if strings.Contains(req.ScenarioId, "expensive") {
		price = 5000 // Test expensive pricing scenario
		log.Printf("   💡 Special scenario detected: Expensive! Price set to %d Kano-Coins", price)
	} else if strings.Contains(req.ScenarioId, "short") {
		duration = 30 // Test short duration scenario
		log.Printf("   💡 Special scenario detected: Short! Duration set to %d minutes", duration)
	}

	resp := &profilev1.ScenarioSnapshotResponse{
		Price:           price,
		DurationMinutes: duration,
	}

	log.Printf("🚀 [gRPC Profile] Returning Price: %d Kano-Coins, Duration: %d mins", resp.Price, resp.DurationMinutes)
	return resp, nil
}


// -----------------------------------------------------------------------------
// FINANCE SERVICE MOCK
// -----------------------------------------------------------------------------
type mockFinanceServer struct {
	financev1.UnimplementedFinanceServiceServer
}

func (s *mockFinanceServer) FreezeCoin(ctx context.Context, req *financev1.FreezeCoinRequest) (*financev1.FinanceCommandResponse, error) {
	log.Printf("📥 [gRPC Finance] Received FreezeCoin for UserID: %s, Amount: %d, Type: %v", req.UserId, req.Amount, req.Type)

	// Dynamic Mock Behavior for testing failure scenarios
	// If UserID or Type indicates a failure test, return failure
	if strings.Contains(req.UserId, "fail") || req.Amount > 4000 {
		log.Printf("   ⚠️ Simulating INSUFFICIENT FUNDS for User: %s", req.UserId)
		return &financev1.FinanceCommandResponse{
			Status:  "FAILED",
			Message: "INSUFFICIENT_FUNDS: Client does not have enough Kano-Coins in wallet",
		}, nil
	}

	resp := &financev1.FinanceCommandResponse{
		Status:  "SUCCESS",
		Message: "[MOCK] Coin successfully frozen for booking reservation",
	}

	log.Printf("🚀 [gRPC Finance] Returning Status: %s (%s)", resp.Status, resp.Message)
	return resp, nil
}


func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	fmt.Println("==========================================================")
	fmt.Println("       RENT-A-GIRLFRIEND GRPC MOCK SERVICES SERVER         ")
	fmt.Println("   (Mocking Profile Service & Finance Service for Local)   ")
	fmt.Println("==========================================================")

	// 1. Start Profile Service Mock (Port 50052)
	lisProfile, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen on port 50052: %v", err)
	}
	grpcProfileServer := grpc.NewServer()
	profilev1.RegisterProfileServiceServer(grpcProfileServer, &mockProfileServer{})

	go func() {
		log.Println("🟢 [gRPC] Profile Service Mock listening on :50052")
		if err := grpcProfileServer.Serve(lisProfile); err != nil {
			log.Fatalf("Profile server failed: %v", err)
		}
	}()

	// 2. Start Finance Service Mock (Port 50053)
	lisFinance, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen on port 50053: %v", err)
	}
	grpcFinanceServer := grpc.NewServer()
	financev1.RegisterFinanceServiceServer(grpcFinanceServer, &mockFinanceServer{})

	log.Println("🟢 [gRPC] Finance Service Mock listening on :50053")
	log.Println("----------------------------------------------------------")
	log.Println("💡 Tip: You can now run your main booking-service!")
	log.Println("   - Use ScenarioID with 'expensive' to test high prices.")
	log.Println("   - Use ClientID/UserID with 'fail' to test insufficient funds.")
	log.Println("----------------------------------------------------------")

	if err := grpcFinanceServer.Serve(lisFinance); err != nil {
		log.Fatalf("Finance server failed: %v", err)
	}
}
