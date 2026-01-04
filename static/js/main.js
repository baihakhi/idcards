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
let statusVal = ""

// getUserbyNik get user detail by NIK inputed
// update action and method to /update if user exist
function getUserbyNik() {
  const nik = document.getElementById("nik").value;
  const userStatus = document.getElementById("dropdown").value;
  switch (userStatus) {
    case "V":
      statusVal = "Vendor"
      break;

    default:
      statusVal = "Penyetor"
      break;
  }

  if (nik.trim() === "") return;

  fetch(`/get?nik=${encodeURIComponent(nik)}`)
    .then((res) => res.json())
    .then((data) => {
      if (data.Error) {
        showWarning("⚠️ " + data.Error);
      } else {
        const user = data.Data;
        const form = document.querySelector("form");
        const formBtn = document.getElementById("formSubmit")

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
        loadCanvas(user.Photo)

        form.setAttribute("action", "/update");
        formBtn.textContent = "Update " + statusVal;
      }
    })
    .catch((err) => {
      console.error("Error checking NIK:", err);
    });
}

// getUserIDbyStatus get current userID by status value
function getUserIDbyStatus() {
  const userStatus = document.getElementById("dropdown").value;
  switch (userStatus) {
    case "V":
      statusVal = "Vendor"
      break;
    default:
      statusVal = "Penyetor"
      break;
  }

  fetch(`/get-id?status=${encodeURIComponent(userStatus)}`)
    .then((res) => res.json())
    .then((data) => {

      if (data.Error) {
        showWarning("⚠️ " + data.Error);
      } else {
        const userID = data.Data;
        console.log("get ", userID);

        clearWarning();
        document.getElementById("userId").textContent = userID;
        document.getElementById("userIdInput").value = userID;
        document.getElementById("formSubmit").textContent = "Simpan " + statusVal
      }
    })
    .catch((err) => {
      console.error("Error generating user ID:", err);
    });
}

function capture() {
  const video = document.getElementById("video");
  const canvas = document.getElementById("canvas");
  const photoInput = document.getElementById("photoData");
  const toggleCam = document.getElementById("camera");
  const ctx = canvas.getContext("2d");
  video.pause();

  canvas.style.display = "none";
  video.style.display = "block";
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

  ctx.drawImage(video, 0, 0, canvas.width, canvas.height);
  photoInput.value = canvas.toDataURL("image/png");
}

function loadCanvas(imgSrc) {

  const canvas = document.getElementById("canvas");
  const photoInput = document.getElementById("photoData");
  const ctx = canvas.getContext('2d');
  const img = new Image();

  img.crossOrigin = "anonymous";
  img.src = "http://localhost:8080/"+imgSrc;
  img.onload = function () {
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
    try {
      const dataURL = canvas.toDataURL("image/png");
      console.log("Data URL length:", dataURL.length);
      photoInput.value = dataURL;
    } catch (err) {
      console.error("Tainted canvas?", err);
    }
    canvas.style.display = "block";
  };

  img.onerror = function (e) {
      console.error('Failed to load the image.', e);
  };

  document.getElementById("video").style.display = "none";
}

function showWarning(message) {
  document.getElementById("warning").style.display = "block";
  document.getElementById("warning").style.position = "absolute";
  document.getElementById("warning").textContent = message;
}

function clearWarning() {
  document.getElementById("warning").style.display = "none";
  document.getElementById("warning").textContent = "";
}

function closePopup() {
  document.getElementById("popup").style.display = "none";
}

function downloadGeneratedFile(userID, fileType) {
  const url = `/download?uid=${encodeURIComponent(userID)}&type=${encodeURIComponent(fileType)}`;
  fetch(url)
    .then((response) => {
      if (!response.ok) {
        throw new Error("Network response was not ok");
      }
      return response.blob();
    })

  window.location.href = url;
}