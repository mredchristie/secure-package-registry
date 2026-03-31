<script lang="ts">
  import { page } from "$app/state";
  import { enhance } from "$app/forms";

  const user = $derived(page.data.user);
</script>

<main>
  <div class="profile-card">
    <h1 class="profile-title">Profile</h1>

    {#if user}
      <div class="info-section">
        <div class="info-row">
          <span class="info-label">Name</span>
          <span class="info-value">{user.name || "—"}</span>
        </div>
        <div class="info-row">
          <span class="info-label">Email</span>
          <span class="info-value">{user.email}</span>
        </div>
        <div class="info-row">
          <span class="info-label">Email Verified</span>
          <span class="info-value">{user.emailVerified ? "Yes" : "No"}</span>
        </div>
        <div class="info-row">
          <span class="info-label">Member Since</span>
          <span class="info-value">
            {new Date(user.createdAt).toLocaleDateString()}
          </span>
        </div>
      </div>

      <div class="actions">
        <form method="POST" action="/logout" use:enhance>
          <button type="submit" class="btn-logout">Log Out</button>
        </form>
      </div>
    {/if}
  </div>
</main>

<style>
  main {
    display: flex;
    align-items: flex-start;
    justify-content: center;
    min-height: calc(100vh - 4rem);
    padding: 3rem 1rem;
  }

  .profile-card {
    width: 100%;
    max-width: 28rem;
    border-radius: 12px;
    border: 1px solid var(--card-border);
    background: var(--card-bg);
    padding: 2rem;
  }

  .profile-title {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 1.5rem;
  }

  .info-section {
    display: flex;
    flex-direction: column;
    gap: 0;
  }

  .info-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--border);
  }

  .info-row:last-child {
    border-bottom: none;
  }

  .info-label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .info-value {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text-primary);
  }

  .actions {
    margin-top: 1.5rem;
    display: flex;
    justify-content: flex-end;
  }

  .btn-logout {
    padding: 0.5rem 1.25rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #dc2626;
    background: rgba(220, 38, 38, 0.08);
    border: 1px solid rgba(220, 38, 38, 0.2);
    border-radius: 6px;
    text-decoration: none;
    transition:
      background 0.15s,
      border-color 0.15s;
  }

  .btn-logout:hover {
    background: rgba(220, 38, 38, 0.15);
    border-color: rgba(220, 38, 38, 0.4);
  }
</style>
