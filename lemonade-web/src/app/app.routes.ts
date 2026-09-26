import { Routes } from '@angular/router';

import { authGuard } from './core/auth.guard';
import { AchievementsComponent } from './pages/achievements/achievements.component';
import { EmpireComponent } from './pages/empire/empire.component';
import { GameComponent } from './pages/game/game.component';
import { HomeComponent } from './pages/home/home.component';
import { RunComponent } from './pages/run/run.component';
import { ScoresComponent } from './pages/scores/scores.component';
import { SigninComponent } from './pages/signin/signin.component';
import { UpgradesComponent } from './pages/upgrades/upgrades.component';

export const routes: Routes = [
  { path: '', component: HomeComponent, title: 'Lemonade Tycoon' },
  { path: 'signin', component: SigninComponent, title: 'Sign in · Lemonade Tycoon' },
  { path: 'game', component: GameComponent, canActivate: [authGuard], title: 'Lemonade Tycoon' },
  {
    path: 'scores',
    component: ScoresComponent,
    canActivate: [authGuard],
    title: 'Scores · Lemonade Tycoon',
  },
  {
    path: 'empire',
    component: EmpireComponent,
    canActivate: [authGuard],
    title: 'Empire · Lemonade Tycoon',
  },
  {
    path: 'upgrades',
    component: UpgradesComponent,
    canActivate: [authGuard],
    title: 'Upgrades · Lemonade Tycoon',
  },
  {
    path: 'achievements',
    component: AchievementsComponent,
    canActivate: [authGuard],
    title: 'Achievements · Lemonade Tycoon',
  },
  {
    path: 'runs/:id',
    component: RunComponent,
    canActivate: [authGuard],
    title: 'Run · Lemonade Tycoon',
  },
  { path: '**', redirectTo: '' },
];
