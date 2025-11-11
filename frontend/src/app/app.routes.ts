import { Routes } from '@angular/router';
import { LayoutComponent } from './components/layout/layout.component';
import { DashboardComponent } from './components/dashboard/dashboard.component';
import { ScanManagerComponent } from './components/scan-manager/scan-manager.component';
import { ScanResultComponent } from './components/scan-result/scan-result.component';
import { SettingsComponent } from './components/settings/settings.component';
import { AiInsightsComponent } from './components/ai-insights/ai-insights.component';
import { AssetsComponent } from './components/assets/assets.component';
import { SchedulerComponent } from './components/scheduler/scheduler.component';

export const routes: Routes = [
  {
    path: '',
    component: LayoutComponent,
    children: [
      { path: '', redirectTo: 'dashboard', pathMatch: 'full' },
      { path: 'dashboard', component: DashboardComponent },
      { path: 'scan-manager', component: ScanManagerComponent },
      { path: 'scan-result/:id', component: ScanResultComponent },
      { path: 'assets', component: AssetsComponent },
      { path: 'scheduler', component: SchedulerComponent },
      { path: 'ai-insights', component: AiInsightsComponent },
      { path: 'settings', component: SettingsComponent },
      { path: '**', redirectTo: 'dashboard' }
    ]
  }
];
