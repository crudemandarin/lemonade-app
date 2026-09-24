import { Routes } from '@angular/router';

import { authGuard } from './core/auth.guard';
import { GameComponent } from './pages/game/game.component';
import { HomeComponent } from './pages/home/home.component';
import { SigninComponent } from './pages/signin/signin.component';

export const routes: Routes = [
  { path: '', component: HomeComponent, title: 'Lemonade Tycoon' },
  { path: 'signin', component: SigninComponent, title: 'Sign in · Lemonade Tycoon' },
  { path: 'game', component: GameComponent, canActivate: [authGuard], title: 'Lemonade Tycoon' },
  { path: '**', redirectTo: '' },
];
