<script lang="ts">
  import { goto } from "$app/navigation";
  import { authClient } from "$lib/client";

  let activeTab: "login" | "register" = $state("login");

  // Login state
  let loginEmail = $state("");
  let loginPassword = $state("");
  let loginError = $state("");
  let loginLoading = $state(false);

  // Register state
  let name = $state("");
  let registerEmail = $state("");
  let registerPassword = $state("");
  let registerError = $state("");
  let registerLoading = $state(false);

  async function handleLogin(e: Event) {
    e.preventDefault();
    loginError = "";
    loginLoading = true;

    const { error: authError } = await authClient.signIn.email({
      email: loginEmail,
      password: loginPassword,
    });

    if (authError) {
      loginError = authError.message ?? "Sign in failed";
      loginLoading = false;
      return;
    }

    await goto("/", { invalidateAll: true });
  }

  async function handleRegister(e: Event) {
    e.preventDefault();
    registerError = "";
    registerLoading = true;

    const { error: authError } = await authClient.signUp.email({
      name,
      email: registerEmail,
      password: registerPassword,
    });

    if (authError) {
      registerError = authError.message ?? "Sign up failed";
      registerLoading = false;
      return;
    }

    // Auto-login after successful registration
    const { error: loginErr } = await authClient.signIn.email({
      email: registerEmail,
      password: registerPassword,
    });

    if (loginErr) {
      // If auto-login fails, switch to login tab
      activeTab = "login";
      loginEmail = registerEmail;
      loginError = "Account created! Please sign in.";
    } else {
      await goto("/", { invalidateAll: true });
    }
    registerLoading = false;
  }
</script>

<main>
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
        Create Account
      </button>
    </div>

    {#if activeTab === "login"}
      <div class="form-section">
        <h1>Welcome back</h1>
        <p class="subtitle">Sign in to your SPR account</p>

        {#if loginError}
          <div
            class="alert"
            class:alert-success={loginError.includes("created")}
            class:alert-error={!loginError.includes("created")}
          >
            {loginError}
          </div>
        {/if}

        <form onsubmit={handleLogin}>
          <div class="form-field">
            <label class="form-label" for="login-email">Email</label>
            <input
              id="login-email"
              type="email"
              bind:value={loginEmail}
              placeholder="you@example.com"
              required
              class="form-input"
            />
          </div>

          <div class="form-field">
            <label class="form-label" for="login-password">Password</label>
            <input
              id="login-password"
              type="password"
              bind:value={loginPassword}
              placeholder="Your password"
              required
              minlength="8"
              class="form-input"
            />
          </div>

          <button type="submit" class="btn-primary" disabled={loginLoading}>
            {loginLoading ? "Signing in..." : "Sign In"}
          </button>
        </form>
      </div>
    {:else}
      <div class="form-section">
        <h1>Create account</h1>
        <p class="subtitle">Get started with secure package verification</p>

        {#if registerError}
          <div class="alert alert-error">{registerError}</div>
        {/if}

        <form onsubmit={handleRegister}>
          <div class="form-field">
            <label class="form-label" for="name">Name</label>
            <input
              id="name"
              type="text"
              bind:value={name}
              placeholder="Your name"
              required
              class="form-input"
            />
          </div>

          <div class="form-field">
            <label class="form-label" for="register-email">Email</label>
            <input
              id="register-email"
              type="email"
              bind:value={registerEmail}
              placeholder="you@example.com"
              required
              class="form-input"
            />
          </div>

          <div class="form-field">
            <label class="form-label" for="register-password">Password</label>
            <input
              id="register-password"
              type="password"
              bind:value={registerPassword}
              placeholder="At least 8 characters"
              required
              minlength="8"
              class="form-input"
            />
          </div>

          <button type="submit" class="btn-primary" disabled={registerLoading}>
            {registerLoading ? "Creating account..." : "Create Account"}
          </button>
        </form>
      </div>
    {/if}
  </div>
</main>

<style>
  main {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: calc(100vh - 4rem);
    padding: 2rem 1rem;
  }

  .auth-card {
    width: 100%;
    max-width: 24rem;
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    overflow: hidden;
  }

  .tab-bar {
    display: flex;
    border-bottom: 1px solid var(--border);
  }

  .tab-btn {
    flex: 1;
    padding: 1rem;
    background: none;
    border: none;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.15s;
  }

  .tab-btn:hover {
    color: var(--text-primary);
    background: var(--bg-secondary);
  }

  .tab-btn.active {
    color: var(--accent);
    border-bottom: 2px solid var(--accent);
    margin-bottom: -1px;
  }

  .form-section {
    padding: 1.5rem 2rem 2rem;
  }

  h1 {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 0.25rem;
  }

  .subtitle {
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-bottom: 1.5rem;
  }

  .form-field {
    display: flex;
    flex-direction: column;
    margin-bottom: 1rem;
  }

  .form-label {
    margin-bottom: 0.375rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-primary);
  }

  .form-input {
    height: 2.5rem;
    width: 100%;
    padding: 0 0.75rem;
    font-size: 0.875rem;
    border-radius: 6px;
    border: 1px solid var(--border);
    background: var(--bg-secondary);
    color: var(--text-primary);
    outline: none;
    transition: border-color 0.15s;
  }

  .form-input::placeholder {
    color: var(--text-secondary);
  }

  .form-input:focus {
    border-color: var(--accent);
    box-shadow: 0 0 0 2px rgba(29, 78, 216, 0.15);
  }

  .btn-primary {
    width: 100%;
    padding: 0.625rem 1rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: #fff;
    background: var(--accent);
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s;
    margin-top: 0.5rem;
  }

  .btn-primary:hover:not(:disabled) {
    background: var(--accent-hover);
  }

  .btn-primary:disabled {
    opacity: 0.55;
    cursor: not-allowed;
  }

  .alert {
    padding: 0.75rem;
    border-radius: 6px;
    font-size: 0.875rem;
    margin-bottom: 1rem;
  }

  .alert-error {
    background: rgba(220, 38, 38, 0.08);
    border: 1px solid rgba(220, 38, 38, 0.2);
    color: #dc2626;
  }

  .alert-success {
    background: rgba(34, 197, 94, 0.08);
    border: 1px solid rgba(34, 197, 94, 0.2);
    color: #16a34a;
  }
</style>
