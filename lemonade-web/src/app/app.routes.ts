import { Routes } from '@angular/router';

import { authGuard, guestGuard, onboardingGuard, signedOutGuard } from './core/auth.guard';
import { AchievementsComponent } from './pages/achievements/achievements.component';
import { GameComponent } from './pages/game/game.component';
import { HomeComponent } from './pages/home/home.component';
import { RunComponent } from './pages/run/run.component';
import { ScoresComponent } from './pages/scores/scores.component';
import { SecureComponent } from './pages/secure/secure.component';
import { SigninComponent } from './pages/signin/signin.component';
import { UsernameComponent } from './pages/username/username.component';

export const routes: Routes = [
  { path: '', component: HomeComponent, title: 'Lemonade Tycoon' },
  {
    path: 'signin',
    component: SigninComponent,
    canActivate: [signedOutGuard],
    title: 'Sign in · Lemonade Tycoon',
  },
  {
    path: 'signin/username',
    component: UsernameComponent,
    canActivate: [onboardingGuard],
    title: 'Choose a username · Lemonade Tycoon',
  },
  {
    path: 'secure',
    component: SecureComponent,
    canActivate: [guestGuard],
    title: 'Secure your account · Lemonade Tycoon',
  },
  { path: 'game', component: GameComponent, canActivate: [authGuard], title: 'Lemonade Tycoon' },
  {
    path: 'scores',
    component: ScoresComponent,
    canActivate: [authGuard],
    title: 'Scores · Lemonade Tycoon',
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
