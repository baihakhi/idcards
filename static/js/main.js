navigator.mediaDevices
  .getUserMedia({
    video: {
      aspectRatio: 3 / 4,
      width: { ideal: 330 },
      height: { ideal: 440 },
    },
  })
  .then((stream) => (document.getElementById("video").srcObject = stream))
  .catch(console.error);

let camPlayed = false;

// getUserbyNik get user detail by NIK inputed
// update action and method to /update if user exist
function getUserbyNik() {
  const nik = document.getElementById("nik").value;
  console.log("send", nik);

  if (nik.trim() === "") return;

  fetch(`/get?nik=${encodeURIComponent(nik)}`)
    .then((res) => res.json())
    .then((data) => {
      if (data.Error) {
        showWarning("⚠️ " + data.Error);
      } else {
        const user = data.Data;
        const form = document.querySelector("form");
        console.log("user id: ", user.ID);

        clearWarning();
        document.getElementById("userIdInput").value = user.ID;
        document.getElementById("userId").textContent = user.ID || "";
        document.querySelector('input[name="userIdInput"]').value =
          user.ID || "";
        document.querySelector('input[name="name"]').value = user.Name || "";
        document.querySelector('input[name="phone"]').value = user.Phone || "";
        document.querySelector('input[name="address"]').value =
          user.Address || "";
        document.querySelector('input[name="rating"]').value =
          user.Rating || "";
        document.querySelector('input[name="notes"]').value = user.Notes || "";

        form.setAttribute("action", "/update");
      }
    })
    .catch((err) => {
      console.error("Error checking NIK:", err);
    });
}

// getUserIDbyStatus get current userID by status value
function getUserIDbyStatus() {
  userStatus = document.getElementById("dropdown").value;

  fetch(`/get-id?status=${encodeURIComponent(userStatus)}`)
    .then((res) => res.json())
    .then((data) => {
      console.log("error", data.Error);

      if (data.Error) {
        showWarning("⚠️ " + data.Error);
      } else {
        const userID = data.Data;
        console.log("get ", userID);

        clearWarning();
        document.getElementById("userId").textContent = userID;
        document.getElementById("userIdInput").value = userID;
      }
    })
    .catch((err) => {
      console.error("Error generating user IDK:", err);
    });
}

function capture() {
  const video = document.getElementById("video");
  const canvas = document.getElementById("canvas");
  const photoInput = document.getElementById("photoData");
  const toggleCam = document.getElementById("camera");
  video.pause();

  canvas.width = 330;
  canvas.height = 450;

  if (!camPlayed) {
    video.play();
    toggleCam.textContent = "📸 Foto";
    camPlayed = !camPlayed;
  } else {
    video.pause();
    toggleCam.textContent = "🔃 Ulang";
    camPlayed = !camPlayed;
  }

  const ctx = canvas.getContext("2d");
  ctx.drawImage(video, 0, 0, canvas.width, canvas.height);

  photoInput.value = canvas.toDataURL("image/png");
}

function showWarning(message) {
  document.getElementById("warning").style.display = "absolute";
  document.getElementById("warning").textContent = message;
}

function clearWarning() {
  document.getElementById("warning").style.display = "none";
}
