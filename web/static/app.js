const form = document.getElementById("shorten-form");
const urlInput = document.getElementById("url-input");
const result = document.getElementById("result");
const shortUrl = document.getElementById("short-url");
const copyBtn = document.getElementById("copy-btn");
const error = document.getElementById("error");

const setError = (message) => {
  error.textContent = message;
  error.classList.toggle("hidden", !message);
};

const setResult = (url) => {
  shortUrl.textContent = url;
  shortUrl.href = url;
  result.classList.toggle("hidden", !url);
};

form.addEventListener("submit", async (event) => {
  event.preventDefault();
  setError("");
  setResult("");

  const url = urlInput.value.trim();
  if (!url) {
    setError("Введите корректный URL.");
    return;
  }

  try {
    const response = await fetch("/shorten", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ url }),
    });

    if (!response.ok) {
      const message = await response.text();
      setError(message || "Не удалось создать короткую ссылку.");
      return;
    }

    const data = await response.json();
    setResult(data.short_url);
  } catch (err) {
    setError("Сервис временно недоступен. Попробуйте позже.");
  }
});

copyBtn.addEventListener("click", async () => {
  if (!shortUrl.textContent) return;
  try {
    await navigator.clipboard.writeText(shortUrl.textContent);
    copyBtn.textContent = "Скопировано!";
    setTimeout(() => {
      copyBtn.textContent = "Скопировать";
    }, 1500);
  } catch (err) {
    setError("Не удалось скопировать ссылку.");
  }
});
