import { Routes } from '@angular/router';
import { AppLayoutComponent } from './layout/app-layout/app-layout.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { ScanManagerComponent } from './pages/scans/scan-manager/scan-manager.component';
import { ScanResultComponent } from './pages/scans/scan-result/scan-result.component';
import { SettingsComponent } from './pages/settings/settings.component';

export const routes: Routes = [
  {
    path: '',
    component: AppLayoutComponent,
    children: [
      { path: '', component: DashboardComponent },
      { path: 'scans', component: ScanManagerComponent },
      { path: 'scans/:id', component: ScanResultComponent },
      { path: 'settings', component: SettingsComponent },
      { path: '**', redirectTo: '' }
    ]
  }
];
