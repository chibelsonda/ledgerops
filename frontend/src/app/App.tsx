import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { LoadingState } from '@/shared/ui/ViewState'
import { LoginPage } from '@/features/auth/LoginPage'
import { useAuth } from '@/features/auth/AuthProvider'
import { AppShell } from './AppShell'

const OverviewPage = lazy(() => import('@/features/dashboard/OverviewPage').then((module) => ({ default: module.OverviewPage })))
const InvoicesPage = lazy(() => import('@/features/invoices/InvoicesPage').then((module) => ({ default: module.InvoicesPage })))
const ReviewPage = lazy(() => import('@/features/reconciliation/ReviewPage').then((module) => ({ default: module.ReviewPage })))
const SyncPage = lazy(() => import('@/features/sync/SyncPage').then((module) => ({ default: module.SyncPage })))
const ClientsPage = lazy(() => import('@/features/clients/ClientsPage').then((module) => ({ default: module.ClientsPage })))
const RemindersPage = lazy(() => import('@/features/reminders/RemindersPage').then((module) => ({ default: module.RemindersPage })))
const ProjectsPage = lazy(() => import('@/features/projects/ProjectsPage').then((module) => ({ default: module.ProjectsPage })))
const ClosePage = lazy(() => import('@/features/close/ClosePage').then((module) => ({ default: module.ClosePage })))
const ForecastPage = lazy(() => import('@/features/forecast/ForecastPage').then((module) => ({ default: module.ForecastPage })))

export default function App() {
  const { session, isLoading } = useAuth()
  if (isLoading) return <LoadingState label="Restoring your secure session" />
  if (!session) return <LoginPage />
  const canOperateSync = session.user.role === 'admin' || session.user.role === 'controller'

  return (
    <Suspense fallback={<LoadingState />}>
      <Routes>
        <Route element={<AppShell />}>
          <Route index element={<OverviewPage />} />
          <Route path="invoices" element={<InvoicesPage />} />
          <Route path="clients" element={<ClientsPage />} />
          <Route path="projects" element={<ProjectsPage />} />
          <Route path="close" element={<ClosePage />} />
          <Route path="forecast" element={<ForecastPage />} />
          <Route path="collections" element={<RemindersPage />} />
          <Route path="review" element={<ReviewPage />} />
          <Route path="sync" element={canOperateSync ? <SyncPage /> : <Navigate to="/" replace />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Route>
      </Routes>
    </Suspense>
  )
}
