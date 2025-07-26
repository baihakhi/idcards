document
  .getElementById("uploadForm")
  .addEventListener("submit", async (event) => {
    event.preventDefault();
    const fileInput = document.getElementById("userFile");
    const file = fileInput.files[0];

    if (!file) {
      alert("Please select a file!");
      return;
    }

    const formData = new FormData();
    formData.append("file", file);

    try {
      const response = await fetch("/upload/upsert", {
        method: "POST",
        body: formData,
      });

      if (response.ok) {
        const result = await response.json();
        alert("File uploaded successfully: " + result.message);
      } else {
        alert("Failed to upload file.");
      }
    } catch (error) {
      console.error("Error uploading file:", error);
      alert("An error occurred while uploading the file.");
    }
  });
