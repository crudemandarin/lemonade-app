import { Component } from '@angular/core';
import { RouterLink, RouterLinkActive } from '@angular/router';

/** The tabs of the current run: the game and the pages that belong to it. */
@Component({
  selector: 'app-run-nav',
  standalone: true,
  imports: [RouterLink, RouterLinkActive],
  template: `
    <nav class="run-nav" aria-label="This run">
      <a routerLink="/game" routerLinkActive="active" class="tab game-tab">Game</a>
      <a routerLink="/production" routerLinkActive="active" class="tab production-tab"
        >Production</a
      >
      <a routerLink="/empire" routerLinkActive="active" class="tab empire-tab">Empire</a>
      <a routerLink="/upgrades" routerLinkActive="active" class="tab upgrades-tab">Upgrades</a>
    </nav>
  `,
  styles: `
    .run-nav {
      display: flex;
      gap: 0.25rem;
      margin: 0 0 1rem;
      padding-bottom: 0.5rem;
      border-bottom: 1px solid var(--border);
      overflow-x: auto;
    }

    .tab {
      padding: 0.375rem 0.75rem;
      border-radius: 999px;
      color: var(--text-muted);
      font-size: 0.9375rem;
      font-weight: 500;
      text-decoration: none;
      white-space: nowrap;

      &:hover {
        color: var(--text);
      }

      &.active {
        background: var(--surface-2, var(--border));
        color: var(--text);
      }
    }
  `,
})
export class RunNavComponent {}
