<script lang="ts">
  import { goto } from "$app/navigation";
  import { authClient } from "$lib/client";

  let name = $state("");
  let email = $state("");
  let password = $state("");
  let error = $state("");
  let loading = $state(false);

  async function handleSubmit(e: Event) {
    e.preventDefault();
    error = "";
    loading = true;

    const { error: authError } = await authClient.signUp.email({
      name,
      email,
      password,
    });

    if (authError) {
      error = authError.message ?? "Sign up failed. Please try again.";
      loading = false;
      return;
    }

    await goto("/", { invalidateAll: true });
  }
</script>

<main>
  <div class="auth-card">
    <h1 class="auth-title">Sign Up</h1>
    <p class="auth-subtitle">Create your SPR account</p>

    {#if error}
      <div class="alert alert-error">{error}</div>
    {/if}

    <form onsubmit={handleSubmit}>
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
        <label class="form-label" for="email">Email</label>
        <input
          id="email"
          type="email"
          bind:value={email}
          placeholder="you@example.com"
          required
          class="form-input"
        />
      </div>

      <div class="form-field">
        <label class="form-label" for="password">Password</label>
        <input
          id="password"
          type="password"
          bind:value={password}
          placeholder="At least 8 characters"
          required
          minlength="8"
          class="form-input"
        />
      </div>

      <button type="submit" disabled={loading} class="btn-primary">
        {loading ? "Creating account..." : "Sign Up"}
      </button>
    </form>

    <p class="auth-footer">
      Already have an account? <a href="/login">Log in</a>
    </p>
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
    padding: 2rem;
  }

  .auth-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 0.25rem;
  }

  .auth-subtitle {
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

  .auth-footer {
    text-align: center;
    font-size: 0.875rem;
    color: var(--text-secondary);
    margin-top: 1.5rem;
  }

  .auth-footer a {
    color: var(--accent);
    text-decoration: none;
    font-weight: 500;
  }

  .auth-footer a:hover {
    text-decoration: underline;
  }
</style>
