from flask import Flask, request, jsonify
from pymongo import MongoClient
import sys
import time
import os
from werkzeug.utils import secure_filename
from flask import send_from_directory

app = Flask(__name__)

# MongoDB Connection
mongo_client = MongoClient("mongodb://localhost:27017/", serverSelectionTimeoutMS=2000)
db = mongo_client["lb4a_test_db"]
users_col = db["users"]

@app.route("/health")
def health_check():
    return jsonify({"status": "alive"}), 200



# ==========================================
# File System Configuration
# ==========================================
UPLOAD_FOLDER = "uploads"
# Automatically create the uploads folder if it doesn't exist
os.makedirs(UPLOAD_FOLDER, exist_ok=True)
app.config['UPLOAD_FOLDER'] = UPLOAD_FOLDER

# ---------------------------------------------------------
# ROUTE 5: POST (Multi-File Upload)
# ---------------------------------------------------------
@app.route("/api/files/upload", methods=["POST"])
def upload_files():
    print(f"\n[FLASK] Received MULTI-FILE UPLOAD on {request.path}")
    
    # Check if the 'files' array exists in the form data
    if 'files' not in request.files:
        return jsonify({"status": "error", "message": "No 'files' field in request"}), 400

    # Get the list of files attached to the request
    files = request.files.getlist('files')
    saved_files = []

    for file in files:
        if file.filename == '':
            continue
        # Secure the filename to prevent directory traversal attacks
        filename = secure_filename(file.filename)
        # Save it to the local hard drive
        file.save(os.path.join(app.config['UPLOAD_FOLDER'], filename))
        saved_files.append(filename)

    return jsonify({
        "status": "success",
        "message": f"Successfully uploaded {len(saved_files)} files.",
        "files": saved_files
    }), 201

# ---------------------------------------------------------
# ROUTE 6: GET (Download a File)
# ---------------------------------------------------------
@app.route("/api/files/<filename>", methods=["GET"])
def download_file(filename):
    print(f"\n[FLASK] Received FILE DOWNLOAD request for: {filename}")
    # Serve the binary file back to the user
    return send_from_directory(app.config['UPLOAD_FOLDER'], filename)

# ---------------------------------------------------------
# ROUTE 1: GET All Users (With Query Parameter Support)
# ---------------------------------------------------------
@app.route("/api/users", methods=["GET"])
def get_users():
    # Grab query parameters (e.g., ?role=admin)
    role_filter = request.args.get("role")
    print(f"\n[FLASK] Received GET request on {request.path} | Query Params: {request.args}")
    
    time.sleep(0.1) 
    
    query = {}
    if role_filter:
        query["role"] = role_filter # Filter MongoDB if the parameter exists
        
    users = list(users_col.find(query, {"_id": 0}))
    
    return jsonify({
        "status": "success",
        "filter_applied": role_filter,
        "count": len(users),
        "data": users
    })

# ---------------------------------------------------------
# ROUTE 2: GET Single User (Dynamic Path Parameter)
# ---------------------------------------------------------
@app.route("/api/users/<user_id>", methods=["GET"])
def get_single_user(user_id):
    print(f"\n[FLASK] Received GET request for DYNAMIC ID: {user_id}")
    time.sleep(0.1)
    
    user = users_col.find_one({"user": user_id}, {"_id": 0})
    
    if user:
        return jsonify({"status": "success", "data": user})
    return jsonify({"status": "error", "message": "User not found"}), 404

# ---------------------------------------------------------
# ROUTE 3: POST (Create User)
# ---------------------------------------------------------
@app.route("/api/users", methods=["POST"])
def create_user():
    data = request.json
    print(f"\n[FLASK] Received POST request on {request.path} | Payload: {data}")
    time.sleep(0.2)
    users_col.insert_one(data.copy())
    return jsonify({"status": "success", "message": "User created", "data": data}), 201

# ---------------------------------------------------------
# ROUTE 4: DELETE (Heavy DB Op to test timeouts)
# ---------------------------------------------------------
@app.route("/api/users/<user_id>", methods=["DELETE"])
def delete_user(user_id):
    print(f"\n[FLASK] Received DELETE request for user: {user_id}")
    time.sleep(0.6) # 600ms latency to test your Gateway Timeout
    users_col.delete_one({"user": user_id})
    return jsonify({"status": "success", "message": f"User {user_id} deleted"})

if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 8081  
    print(f"🚀 Flask Server running on port {port} with MongoDB connected.")
    app.run(host="0.0.0.0", port=port)
