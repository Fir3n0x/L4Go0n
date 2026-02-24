// duration (in ms)
const FADE_OUT_DELAY    = 500;   // lag before fade-out splash
const TOTAL_SPLASH_TIME = 1500;  // lag total before login display

window.addEventListener("load", () => {
  const intro    = document.getElementById("intro-animation");
  const loginBox = document.getElementById("login-box");
  const hasError = !!loginBox.querySelector("p");
  const introDone= sessionStorage.getItem("introDone") === "1";

  // if already done, skip splash
  if (introDone || hasError) {
    intro.style.display = "none";
    loginBox.classList.remove("hidden");
    return;
  }

  // Splash fade-in
  setTimeout(() => {
    intro.style.opacity = "1";
  }, 50);

  // Fade-out splash
  setTimeout(() => {
    intro.style.opacity = "0";
  }, FADE_OUT_DELAY);

  // Delete splash and show login
  setTimeout(() => {
    intro.remove();
    loginBox.classList.remove("hidden");
    sessionStorage.setItem("introDone", "1");
  }, TOTAL_SPLASH_TIME);
});
