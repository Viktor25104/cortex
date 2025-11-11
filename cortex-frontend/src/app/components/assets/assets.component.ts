import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-assets',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="placeholder-page">
      <div class="placeholder-card">
        <div class="placeholder-icon">📦</div>
        <h1>Asset Management</h1>
        <p class="subtitle">Inventory and classification of discovered network assets</p>
        <div class="coming-soon">Coming in v7.0</div>
        <p class="description">
          The Asset Management module will provide comprehensive inventory tracking,
          automated classification, and lifecycle management of all discovered network assets.
        </p>
      </div>
    </div>
  `,
  styles: [`
    .placeholder-page { display:flex; align-items:center; justify-content:center; min-height:80vh; padding:2rem; }
    .placeholder-card { max-width:600px; text-align:center; padding:3rem; background:var(--secondary-bg); border:1px solid rgba(0,174,255,0.2); border-radius:8px; }
    .placeholder-icon { font-size:4rem; margin-bottom:1.5rem; }
    h1 { margin:0 0 .5rem 0; font-size:2rem; color:var(--accent-primary); font-weight:700; }
    .subtitle { margin:0 0 1.5rem 0; font-size:.95rem; color:rgba(204,214,246,.6); text-transform:uppercase; letter-spacing:1px; }
    .coming-soon { display:inline-block; padding:.5rem 1.5rem; background:rgba(0,174,255,.2); color:var(--accent-primary); border-radius:20px; font-size:.875rem; font-weight:600; text-transform:uppercase; letter-spacing:.5px; margin-bottom:1.5rem; }
    .description { margin:0; font-size:1rem; line-height:1.7; color:rgba(204,214,246,.7); }
  `]
})
export class AssetsComponent {}

