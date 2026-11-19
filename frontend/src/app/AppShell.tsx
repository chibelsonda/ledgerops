import { Dialog, DialogBackdrop, DialogPanel, Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/react'
import { useQuery } from '@tanstack/react-query'
import { BriefcaseBusiness, CalendarCheck, ChartLine, ChartNoAxesCombined, ChevronDown, ChevronLeft, ChevronRight, FileText, LogOut, MailCheck, Menu as MenuIcon, RefreshCw, ScanSearch, Users, X } from 'lucide-react'
import { useState } from 'react'
import { NavLink, Outlet } from 'react-router-dom'
import { dashboardApi } from '@/features/dashboard/api'
import { useAuth } from '@/features/auth/AuthProvider'
import type { Role } from '@/features/auth/types'
import { titleCase } from '@/shared/lib/format'
import { capabilities } from '@/shared/lib/capabilities'
import { queryKeys } from '@/shared/api/queryKeys'
import { LedgerOpsMark } from '@/shared/ui/LedgerOpsMark'

interface NavigationItem {
  to: string
  label: string
  icon: typeof ChartNoAxesCombined
  end?: boolean
  allowed?: (role?: Role) => boolean
}

const navigation: NavigationItem[] = [
  { to: '/', label: 'Overview', icon: ChartNoAxesCombined, end: true },
  { to: '/invoices', label: 'Invoices', icon: FileText },
  { to: '/clients', label: 'Clients', icon: Users },
  { to: '/projects', label: 'Projects', icon: BriefcaseBusiness },
  { to: '/close', label: 'Month-end close', icon: CalendarCheck },
  { to: '/forecast', label: 'Cash forecast', icon: ChartLine },
  { to: '/collections', label: 'Collections', icon: MailCheck },
  { to: '/review', label: 'Review', icon: ScanSearch },
  { to: '/sync', label: 'Data sync', icon: RefreshCw, allowed: capabilities.operate },
]

export function AppShell() {
  const { session, logout } = useAuth()
  const { data } = useQuery({ queryKey: queryKeys.dashboard, queryFn: dashboardApi.get })
  const [mobileNavigationOpen, setMobileNavigationOpen] = useState(false)
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  if (!session) return null
  const initials = session.user.displayName.split(/\s+/).map((name) => name[0]).join('').slice(0, 2).toUpperCase()

  const brand = (
    <div className="brand"><span className="brand-mark"><LedgerOpsMark /></span><span>LedgerOps</span></div>
  )
  const navigationLinks = (onNavigate?: () => void) => (
    <nav aria-label="Primary navigation">
      {navigation.filter((item) => !item.allowed || item.allowed(session.user.role)).map(({ to, label, icon: Icon, end }) => (
        <NavLink key={to} to={to} end={end} onClick={onNavigate} aria-label={label} title={sidebarCollapsed ? label : undefined} className={({ isActive }) => isActive ? 'nav-link active' : 'nav-link'}>
          <Icon size={18} strokeWidth={1.8} /><span className="nav-label">{label}</span>
          {label === 'Review' && data?.metrics.openReviewCount ? <span className="nav-count">{data.metrics.openReviewCount}</span> : null}
        </NavLink>
      ))}
    </nav>
  )

  return (
    <div className={sidebarCollapsed ? 'app-shell sidebar-collapsed' : 'app-shell'}>
      <aside className="sidebar" id="desktop-sidebar">
        <div className="sidebar-heading">
          {brand}
          <button
            className="desktop-sidebar-toggle"
            type="button"
            aria-label={sidebarCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}
            aria-controls="desktop-sidebar"
            aria-expanded={!sidebarCollapsed}
            onClick={() => setSidebarCollapsed((collapsed) => !collapsed)}
          >
            {sidebarCollapsed
              ? <ChevronRight size={20} aria-hidden="true" />
              : <ChevronLeft size={20} aria-hidden="true" />}
          </button>
        </div>
        {navigationLinks()}
      </aside>
      <Dialog open={mobileNavigationOpen} onClose={setMobileNavigationOpen} className="mobile-sidebar-dialog">
        <DialogBackdrop className="mobile-sidebar-backdrop" />
        <DialogPanel className="sidebar mobile-sidebar">
          <div className="mobile-sidebar-heading">
            {brand}
            <button className="mobile-sidebar-close" type="button" aria-label="Close navigation" onClick={() => setMobileNavigationOpen(false)}><X size={24} /></button>
          </div>
          {navigationLinks(() => setMobileNavigationOpen(false))}
        </DialogPanel>
      </Dialog>
      <main className="main-content">
        <header className="main-content-header">
          <div className="main-content-header-inner">
            <div className="main-content-header-start">
              <button className="mobile-menu-trigger" type="button" aria-label="Open navigation" onClick={() => setMobileNavigationOpen(true)}><MenuIcon size={21} /></button>
              <strong>Admin Panel</strong>
            </div>
            <Menu as="div" className="user-menu">
              <MenuButton className="user-menu-trigger">
                <span className="avatar">{initials}</span>
                <span className="user-menu-copy"><strong>{session.user.displayName}</strong><small>{titleCase(session.user.role)}</small></span>
                <ChevronDown size={15} aria-hidden="true" />
              </MenuButton>
              <MenuItems anchor={{ to: 'bottom end', gap: 8 }} className="user-menu-dropdown">
                <div className="user-menu-details">
                  <strong>{session.user.displayName}</strong>
                  <span>{session.user.email}</span>
                </div>
                <MenuItem as="button" type="button" className="user-menu-signout" onClick={() => void logout()}>
                  <LogOut size={16} aria-hidden="true" /> Sign out
                </MenuItem>
              </MenuItems>
            </Menu>
          </div>
        </header>
        <Outlet />
      </main>
    </div>
  )
}
