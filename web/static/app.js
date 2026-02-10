const form = document.getElementById("shorten-form");
const urlInput = document.getElementById("url-input");
const result = document.getElementById("result");
const shortUrl = document.getElementById("short-url");
const copyBtn = document.getElementById("copy-btn");
const error = document.getElementById("error");
const authSection = document.getElementById("auth-section");
const profileSection = document.getElementById("profile-section");
const loginForm = document.getElementById("login-form");
const registerForm = document.getElementById("register-form");
const profileEmail = document.getElementById("profile-email");
const profileActions = document.getElementById("profile-actions");
const logoutBtn = document.getElementById("logout-btn");
const linksList = document.getElementById("links-list");
const refreshBtn = document.getElementById("refresh-btn");
const totalLinks = document.getElementById("total-links");
const totalClicks = document.getElementById("total-clicks");

const setError = (message) => {
  error.textContent = message;
  error.classList.toggle("hidden", !message);
};

const setResult = (url) => {
  shortUrl.textContent = url;
  shortUrl.href = url;
  result.classList.toggle("hidden", !url);
};

const toggleAuth = (isAuthenticated) => {
  authSection.classList.toggle("hidden", isAuthenticated);
  profileSection.classList.toggle("hidden", !isAuthenticated);
  profileActions.classList.toggle("hidden", !isAuthenticated);
};

const renderLinks = (links) => {
  linksList.innerHTML = "";
  if (!links.length) {
    linksList.innerHTML = "<p class=\"empty\">Ссылок пока нет.</p>";
    return;
  }
  links.forEach((link) => {
    const item = document.createElement("div");
    item.className = "link-item";
    item.innerHTML = `
      <div class="link-info">
        <a href="${link.short_url}" target="_blank" rel="noreferrer">${link.short_url}</a>
        <span class="link-original">${link.url}</span>
      </div>
      <div class="link-actions">
        <span class="link-clicks">Переходов: ${link.clicks}</span>
        <button class="ghost" data-code="${link.code}">Удалить</button>
      </div>
    `;
    item.querySelector("button").addEventListener("click", () =>
      deleteLink(link.code)
    );
    linksList.appendChild(item);
  });
};

const loadProfile = async () => {
  try {
    const response = await fetch("/me");
    if (!response.ok) {
      toggleAuth(false);
      return;
    }
    const user = await response.json();
    profileEmail.textContent = user.email;
    toggleAuth(true);
    await loadLinks();
  } catch (err) {
    toggleAuth(false);
  }
};

const loadLinks = async () => {
  try {
    const response = await fetch("/links");
    if (!response.ok) {
      throw new Error();
    }
    const links = await response.json();
    const total = links.length;
    const clicks = links.reduce((sum, link) => sum + link.clicks, 0);
    totalLinks.textContent = total;
    totalClicks.textContent = clicks;
    renderLinks(links);
  } catch (err) {
    setError("Не удалось загрузить список ссылок.");
  }
};

const deleteLink = async (code) => {
  try {
    const response = await fetch(`/links/${code}`, { method: "DELETE" });
    if (!response.ok) {
      throw new Error();
    }
    await loadLinks();
  } catch (err) {
    setError("Не удалось удалить ссылку.");
  }
};

loginForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  setError("");
  const email = document.getElementById("login-email").value.trim();
  const password = document.getElementById("login-password").value;
  try {
    const response = await fetch("/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!response.ok) {
      const message = await response.text();
      setError(message || "Не удалось войти.");
      return;
    }
    loginForm.reset();
    await loadProfile();
  } catch (err) {
    setError("Сервис временно недоступен. Попробуйте позже.");
  }
});

registerForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  setError("");
  const email = document.getElementById("register-email").value.trim();
  const password = document.getElementById("register-password").value;
  try {
    const response = await fetch("/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
    });
    if (!response.ok) {
      const message = await response.text();
      setError(message || "Не удалось зарегистрироваться.");
      return;
    }
    registerForm.reset();
    setError("Регистрация успешна. Теперь войдите.");
  } catch (err) {
    setError("Сервис временно недоступен. Попробуйте позже.");
  }
});

logoutBtn.addEventListener("click", async () => {
  try {
    await fetch("/logout", { method: "POST" });
  } finally {
    toggleAuth(false);
    profileEmail.textContent = "";
  }
});

refreshBtn.addEventListener("click", () => {
  loadLinks();
});

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
    urlInput.value = "";
    await loadLinks();
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

loadProfile();
