# C2C Backend API Documentation 

## Authentication Routes

### **1. Sign Up**
**Endpoint:** `POST /api/v1/auth/signup`  
**Description:** Register a new user.  

**Request Body:**
```json
{
  "contact_number": "1234567890",
  "gender": "female",
  "reg_no": "REG2025002",
  "role": "participant",
  "internal": true,
  "college_name": "<only counts if internal is false>"
}

```

---

### **2. Sign In**
**Endpoint:** `GET /api/v1/auth/signin`  
**Description:** Log in an existing user.  

---

### **3. Get User**
**Endpoint:** `GET /api/v1/auth/user`
**Description:** Get user info.  

---

## RSVP Stuff

### **1. Get Dashboard**
**Endpoint:** `GET /api/v1/dashboard`  
**Description:** Get user dahboard.

### **3. Create Submission**
**Endpoint:** `POST /api/v1/team/submission`  
**Description:** Create a new team.
**Request Body:**
```json
{
  "github_url": "https://github.com/example-org/hackathon-project",
  "figma_url": "https://www.figma.com/file/abcd1234/Project-Design",
  "other": "https://example.com/demo",
  "track_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

---



## Team Routes

### **1. Create Team**
**Endpoint:** `POST /api/v1/team/create`  
**Description:** Create a new team.  

**Request Body:**
```json
{
  "name": "Hackathon Avengers",
  "description": "A group of developers aiming to win the hackathon."
}
```
> **Note:** `description` is optional, but `track_id` is required.

---

### **2. Join Team**
**Endpoint:** `POST /api/v1/team/join`  
**Description:** Join an existing team using a team code.  

**Request Body:**
```json
{
  "code": "99GDIV58"
}
```

---

### **3. Leave Team**
**Endpoint:** `GET /api/v1/team/leave`  
**Description:** Leave the current team.  

---

## Track Routes *(Admin Only)*

### **1. Create Track**
**Endpoint:** `POST /api/v1/tracks/create`  
**Description:** Create a new track.  

**Request Body:**
```json
{
  "title": "AI/ML",
  "description": "Projects related to Artificial Intelligence and Machine Learning"
}
```

---

### **2. Get Tracks**
**Endpoint:** `GET /api/v1/tracks/getall`  
**Description:** Retrieve all available tracks.  

---

### **3. Update Track**
**Endpoint:** `PUT /api/v1/tracks/update/:trackid`  
**Description:** Update an existing track.  

**Request Body (any field is optional):**
```json
{
  "title": "AI & Machine Learning",
  "description": "Updated description for AI/ML track"
}
```

---

### **4. Delete Track**
**Endpoint:** `DELETE /api/v1/tracks/delete/:trackid`  
**Description:** Delete a specific track.  

---

## Round Routes *(Admin Only)*

### **1. Create Round**
**Endpoint:** `POST /api/v1/round`  
**Description:** Create a new round.  

**Request Body:**
```json
{
  "name": "Round 1",
  "round_number": 1,
  "description": "Initial qualifying round"
} 
```

---

### **2. Delete Round**
**Endpoint:** `DELETE /api/v1/round/:rno`  
**Description:** Delete a specific round by round number.  

---

### **3. Update Round**
**Endpoint:** `PUT /api/v1/round/:rno`  
**Description:** Update round details.  

**Request Body:**
```json
{
  "name": "Round 1 - Updated",
  "round_number": 1,
  "description": "Updated description for the first round"
}
```

---

### **4. Get Round Rankings**
**Endpoint:** `GET /api/v1/round/:rno`  
**Description:** Retrieve rankings for a specific round.  

---

### **5. Round Promotion**
**Endpoint:** `POST /api/v1/round/:rno/promote`  
**Description:** Promote specific teams to the next round.  

**Request Body:**
```json
{
  "team_ids": [
    "a145bae2-7933-11f0-80ce-f278fa837760",
    "a86d7b48-7933-11f0-80ce-f278fa837760"
  ]
}
```

---

## Review Routes *(Reviewers & Admins Only)*

### **1. Post Review**
**Endpoint:** `POST /api/v1/reviews/post/:rno/:team_id`  
**Description:** Post a new review.  

**Request Body:**
```json
{
  "design": 10,
  "implementation": 9,
  "uniqueness": 8,
  "practicality": 9,
  "comments": "Very impressive work with excellent implementation quality."
}
```

---

### **2. Delete Review**
**Endpoint:** `DELETE /api/v1/reviews/:rno/:team_id`  
**Description:** Delete a review.  

---

### **3. Get All Reviews**
**Endpoint:** `GET /api/v1/reviews/all`  
**Description:** Display all reviews.  

---

## Pro Tips for Usage
- Authentication tokens (if implemented) should be sent via the `Authorization` header with `Bearer <token>` format.
