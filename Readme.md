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
  "reg_no": "REG2025002"
}
```

---

### **2. Sign In**
**Endpoint:** `GET /api/v1/auth/signin`  
**Description:** Log in an existing user.  

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
> **Note:** `description` is optional.

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

##  Round Routes *(Admin Only)*

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

**Pro Tips for Usage**  
- Authentication tokens (if implemented) should be sent via the `Authorization` header with `Bearer <token>` format. 
