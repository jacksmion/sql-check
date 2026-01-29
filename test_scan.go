package main

func main() {
    // This is a dummy go file for testing extraction
    q1 := "SELECT * FROM users WHERE id = 1"
    q2 := `UPDATE orders SET status = 'paid' WHERE id = 100`
    q3 := "DELETE FROM users" // BAD: Unsafe Delete
    
    // Performance & Index tests
    q4 := "SELECT * FROM users WHERE name = 123" // BAD: Implicit Conversion (name is str) + Index Miss
    q5 := "SELECT * FROM users WHERE email LIKE '%@gmail.com'" // BAD: Leading Wildcard
    q6 := "SELECT * FROM users WHERE created_at = '2023-01-01'" // BAD: Index Miss (no index on created_at)
    q7 := "SELECT * FROM users WHERE id > 1 LIMIT 10000, 10" // BAD: Deep Pagination
    
    q8 := "SELECT * FROM users WHERE email = 'test@example.com'" // GOOD: Index Hit
    s := "SELECTING items is fun" 
}
