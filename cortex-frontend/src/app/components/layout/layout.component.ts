import { Component } from '@angular/core';
import { RouterModule } from '@angular/router';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-layout',
  standalone: true,
  imports: [CommonModule, RouterModule],
  templateUrl: './layout.component.html',
  styleUrls: ['./layout.component.scss']
})
export class LayoutComponent {
  navigationItems = [
    { route: '/dashboard', icon: '🏠', label: 'Dashboard' },
    { route: '/scan-manager', icon: '🛰️', label: 'Scan Manager' },
    { route: '/assets', icon: '📦', label: 'Asset Management' },
    { route: '/scheduler', icon: '⏱️', label: 'Scheduler' },
    { route: '/ai-insights', icon: '🧠', label: 'AI Insights' },
    { route: '/settings', icon: '⚙️', label: 'Settings' }
  ];
}

