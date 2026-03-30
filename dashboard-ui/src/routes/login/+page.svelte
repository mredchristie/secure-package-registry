<script lang="ts">
  import { authClient } from "$lib/client";
  import { PUBLIC_HOME_BASE_URL } from "$env/static/public";
  let activeTab: "login" | "register" = $state("login");

  // Login state
  let loginEmail = $state("");
  let loginPassword = $state("");
  let loginError = $state("");
  let loginLoading = $state(false);

  // Register state
  let firstName = $state("");
  let lastName = $state("");
  let registerEmail = $state("");
  let registerPassword = $state("");
  let confirmPassword = $state("");
  let registerError = $state("");
  let registerLoading = $state(false);
  let registerSuccess = $state(false);

  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

  function validateRegister(): string | null {
    if (!emailRegex.test(registerEmail))
      return "Please enter a valid email address";
    if (registerPassword.length < 8)
      return "Password must be at least 8 characters";
    if (!/[A-Z]/.test(registerPassword))
      return "Password must contain at least one uppercase letter";
    if (!/[0-9]/.test(registerPassword))
      return "Password must contain at least one number";
    if (registerPassword !== confirmPassword) return "Passwords do not match";
    return null;
  }

  async function handleLogin(e: Event) {
    e.preventDefault();
    loginLoading = true;
    loginError = "";

    const { data, error } = await authClient.signIn.email({
      email: loginEmail,
      password: loginPassword,
    });

    if (error) {
      loginError = error.message ?? "Sign in failed";
    } else {
      window.location.href = PUBLIC_HOME_BASE_URL;
    }
    loginLoading = false;
  }

  async function handleRegister(e: Event) {
    e.preventDefault();
    const validationError = validateRegister();
    if (validationError) {
      registerError = validationError;
      return;
    }
    registerLoading = true;
    registerError = "";

    const { data, error } = await authClient.signUp.email({
      email: registerEmail,
      password: registerPassword,
      name: `${firstName} ${lastName}`.trim(),
    });

    if (error) {
      registerError = error.message ?? "Sign up failed";
    } else {
      registerSuccess = true;
      activeTab = "login";
    }
    registerLoading = false;
  }
</script>

<div class="auth-page">
  <div class="auth-container">
    <div class="auth-card">
      <div class="tab-bar">
        <button
          class="tab-btn"
          class:active={activeTab === "login"}
          onclick={() => (activeTab = "login")}
        >
          Sign In
        </button>
        <button
          class="tab-btn"
          class:active={activeTab === "register"}
          onclick={() => (activeTab = "register")}
        >
          Register
        </button>
      </div>

      {#if activeTab === "login"}
        <div class="form-section">
          <h1>Welcome back</h1>
          <p class="subtitle">Access your secure package registry</p>

          {#if registerSuccess}
            <p class="success-message">Account created! Please sign in.</p>
          {/if}

          <form onsubmit={handleLogin}>
            <div class="form-group">
              <label for="login-email">Email</label>
              <input
                type="email"
                id="login-email"
                placeholder="you@example.com"
                required
                bind:value={loginEmail}
              />
            </div>

            <div class="form-group">
              <label for="login-password">Password</label>
              <input
                type="password"
                id="login-password"
                placeholder="••••••••"
                required
                bind:value={loginPassword}
              />
            </div>

            {#if loginError}
              <p class="error-message">{loginError}</p>
            {/if}

            <div class="form-options">
              <label class="checkbox">
                <input type="checkbox" name="remember" />
                <span>Remember me</span>
              </label>
            </div>

            <button type="submit" class="submit-button" disabled={loginLoading}>
              {loginLoading ? "Signing in..." : "Sign In"}
            </button>
          </form>

          <p class="switch-prompt">
            Don't have an account?
            <button class="switch-link" onclick={() => (activeTab = "register")}
              >Register</button
            >
          </p>
        </div>
      {:else}
        <div class="form-section">
          <h1>Create account</h1>
          <p class="subtitle">Get started with secure package verification</p>

          <form onsubmit={handleRegister}>
            <div class="form-row">
              <div class="form-group">
                <label for="firstName">First Name</label>
                <input
                  type="text"
                  id="firstName"
                  placeholder="John"
                  required
                  bind:value={firstName}
                />
              </div>
              <div class="form-group">
                <label for="lastName">Last Name</label>
                <input
                  type="text"
                  id="lastName"
                  placeholder="Doe"
                  required
                  bind:value={lastName}
                />
              </div>
            </div>

            <div class="form-group">
              <label for="register-email">Email</label>
              <input
                type="email"
                id="register-email"
                placeholder="you@company.com"
                required
                bind:value={registerEmail}
              />
            </div>

            <div class="form-group">
              <label for="register-password">Password</label>
              <input
                type="password"
                id="register-password"
                placeholder="••••••••"
                required
                bind:value={registerPassword}
              />
              <span class="helper-text"
                >At least 8 characters, one uppercase letter, one number</span
              >
            </div>

            <div class="form-group">
              <label for="confirmPassword">Confirm Password</label>
              <input
                type="password"
                id="confirmPassword"
                placeholder="••••••••"
                required
                bind:value={confirmPassword}
              />
            </div>

            {#if registerError}
              <p class="error-message">{registerError}</p>
            {/if}

            <label class="checkbox terms">
              <input type="checkbox" required />
              <span
                >I agree to the <a href="/terms">Terms of Service</a> and
                <a href="/privacy">Privacy Policy</a></span
              >
            </label>

            <button
              type="submit"
              class="submit-button"
              disabled={registerLoading}
            >
              {registerLoading ? "Creating account..." : "Create Account"}
            </button>
          </form>

          <p class="switch-prompt">
            Already have an account?
            <button class="switch-link" onclick={() => (activeTab = "login")}
              >Sign In</button
            >
          </p>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  .auth-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-primary);
    padding: 2rem;
  }

  .auth-container {
    width: 100%;
    max-width: 480px;
  }

  .auth-card {
    background: var(--card-bg);
    border: 1px solid var(--card-border);
    border-radius: 16px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
    overflow: hidden;
  }

  .tab-bar {
    display: grid;
    grid-template-columns: 1fr 1fr;
    border-bottom: 1px solid var(--card-border);
  }

  .tab-btn {
    padding: 1rem;
    background: none;
    border: none;
    font-size: 0.95rem;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.2s;
  }

  .tab-btn.active {
    color: var(--accent);
    border-bottom: 2px solid var(--accent);
    margin-bottom: -1px;
  }

  .tab-btn:hover:not(.active) {
    color: var(--text-primary);
    background: var(--bg-primary);
  }

  .form-section {
    padding: 2.5rem;
  }

  h1 {
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 0.375rem;
    text-align: center;
  }

  .subtitle {
    color: var(--text-secondary);
    text-align: center;
    margin-bottom: 2rem;
    font-size: 0.9rem;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1rem;
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
  }

  label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  input[type="text"],
  input[type="email"],
  input[type="password"] {
    padding: 0.75rem 1rem;
    border: 1px solid var(--border);
    border-radius: 8px;
    font-size: 0.95rem;
    background: var(--bg-primary);
    color: var(--text-primary);
    transition: all 0.2s;
  }

  input[type="text"]:focus,
  input[type="email"]:focus,
  input[type="password"]:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 3px rgba(79, 195, 247, 0.1);
  }

  input::placeholder {
    color: var(--text-secondary);
    opacity: 0.5;
  }

  .helper-text {
    font-size: 0.8rem;
    color: var(--text-secondary);
  }

  .form-options {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.875rem;
  }

  .checkbox {
    display: flex;
    align-items: flex-start;
    gap: 0.6rem;
    cursor: pointer;
    color: var(--text-secondary);
    font-size: 0.875rem;
    line-height: 1.5;
  }

  .checkbox.terms {
    margin-top: 0.25rem;
  }

  .checkbox input[type="checkbox"] {
    width: 16px;
    height: 16px;
    margin-top: 0.15rem;
    flex-shrink: 0;
    cursor: pointer;
    accent-color: var(--accent);
  }

  .checkbox a {
    color: var(--accent);
    text-decoration: none;
  }

  .checkbox a:hover {
    text-decoration: underline;
  }

  .submit-button {
    background: var(--accent);
    color: var(--bg-primary);
    padding: 0.875rem 2rem;
    border: none;
    border-radius: 8px;
    font-size: 1rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s;
    margin-top: 0.25rem;
  }

  .submit-button:hover:not(:disabled) {
    background: var(--accent-hover);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(79, 195, 247, 0.3);
  }

  .submit-button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-message {
    color: #ef4444;
    font-size: 0.875rem;
    text-align: center;
  }

  .success-message {
    color: #22c55e;
    font-size: 0.875rem;
    text-align: center;
    margin-bottom: 1rem;
  }

  .switch-prompt {
    margin-top: 1.5rem;
    text-align: center;
    color: var(--text-secondary);
    font-size: 0.875rem;
  }

  .switch-link {
    background: none;
    border: none;
    color: var(--accent);
    font-weight: 600;
    cursor: pointer;
    font-size: 0.875rem;
    padding: 0;
    transition: color 0.2s;
  }

  .switch-link:hover {
    color: var(--accent-hover);
  }

  @media (max-width: 520px) {
    .form-section {
      padding: 1.75rem;
    }

    .form-row {
      grid-template-columns: 1fr;
    }
  }
</style>
