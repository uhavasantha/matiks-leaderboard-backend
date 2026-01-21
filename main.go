package main

import (
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// User represents a player in the system
type User struct {
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	Rank     int    `json:"rank"`
}

// LeaderboardSystem holds all data in memory for speed
type LeaderboardSystem struct {
	sync.RWMutex
	Users   []*User          // Sorted list for Leaderboard display
	UserMap map[string]*User // Hash map for O(1) lookups
}

var sys = &LeaderboardSystem{
	UserMap: make(map[string]*User),
}

// RecalculateRanks implements the "Tie-Aware Ranking" requirement [cite: 12, 47]
// Users with the same rating get the same rank.
func (ls *LeaderboardSystem) RecalculateRanks() {
	ls.Lock()
	defer ls.Unlock()

	// Sort Descending by Rating
	sort.Slice(ls.Users, func(i, j int) bool {
		return ls.Users[i].Rating > ls.Users[j].Rating
	})

	// Assign Ranks
	currentRank := 1
	for i, user := range ls.Users {
		// If not first user and rating is different from previous, update rank
		if i > 0 && user.Rating < ls.Users[i-1].Rating {
			currentRank = i + 1
		}
		user.Rank = currentRank
	}
}

// SeedUsers generates 10,000 users as required [cite: 25]
func SeedUsers() {
	sys.Lock()
	defer sys.Unlock()

	names := []string{"rahul", "arjun", "priya", "vikram", "anisha", "rohan", "sara", "kabir"}

	for i := 0; i < 10000; i++ {
		baseName := names[rand.Intn(len(names))]
		username := fmt.Sprintf("%s_%d", baseName, i)
		// Rating between 100 and 5000 [cite: 45]
		rating := rand.Intn(4901) + 100 

		newUser := &User{Username: username, Rating: rating}
		sys.Users = append(sys.Users, newUser)
		sys.UserMap[username] = newUser
	}
	fmt.Println("✅ Seeded 10,000 users.")
}

// StartScoreUpdates simulates random updates every 10s [cite: 28, 57]
func StartScoreUpdates() {
	ticker := time.NewTicker(10 * time.Second)
	go func() {
		for range ticker.C {
			sys.Lock()
			// Update 50 random users to simulate activity
			for k := 0; k < 50; k++ {
				idx := rand.Intn(len(sys.Users))
				sys.Users[idx].Rating = rand.Intn(4901) + 100
			}
			sys.Unlock()

			sys.RecalculateRanks()
			fmt.Println("🔄 Ratings updated and Ranks recalculated")
		}
	}()
}

func main() {
	rand.Seed(time.Now().UnixNano())
	SeedUsers()
	sys.RecalculateRanks()
	StartScoreUpdates()

	r := gin.Default()
	
	// Enable CORS so the frontend can talk to backend
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	r.Use(cors.New(config))

	// API: Get Leaderboard
	r.GET("/leaderboard", func(c *gin.Context) {
		sys.RLock()
		defer sys.RUnlock()

		// Return top 100 to keep UI snappy (Pagination logic)
		limit := 100
		if len(sys.Users) < limit {
			limit = len(sys.Users)
		}
		c.JSON(200, sys.Users[:limit])
	})

	// API: Search User [cite: 54]
	r.GET("/search", func(c *gin.Context) {
		query := strings.ToLower(c.Query("username"))
		if query == "" {
			c.JSON(400, gin.H{"error": "Query required"})
			return
		}

		var results []*User
		sys.RLock()
		defer sys.RUnlock()

		// 1. Exact Match (O(1) Lookup)
		if u, exists := sys.UserMap[query]; exists {
			results = append(results, u)
		}

		// 2. Partial Match (Search first 10k users efficiently)
		count := 0
		for _, u := range sys.Users {
			if strings.Contains(strings.ToLower(u.Username), query) && u.Username != query {
				results = append(results, u)
				count++
			}
			if count >= 10 { // Limit results to prevent lag
				break 
			}
		}

		c.JSON(200, results)
	})

	port := os.Getenv("PORT")
if port == "" {
	port = "8080" // fallback for local run
}

fmt.Printf("🚀 Server running on port %s\n", port)
r.Run(":" + port)


}
