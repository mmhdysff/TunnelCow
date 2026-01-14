import { useState, useEffect } from 'react';
import { AreaChart, Area, ResponsiveContainer, Tooltip } from 'recharts';
import { Activity, Plus, Trash2, ArrowUpRight, Zap, Shield, RefreshCw, Server, Globe, AlertCircle, X, CheckSquare, Square, Eye, Search, Code, Clock, LockOpen } from 'lucide-react';
import clsx from 'clsx';
import { createPortal } from 'react-dom';



const API_BASE = import.meta.env.DEV ? 'http://localhost:10000/api' : '/api';

const BulkDeleteModal = ({ isOpen, progress, total }) => {
  if (!isOpen) return null;
  const percentage = Math.round((progress / total) * 100) || 0;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm transition-all duration-300">
      <div className="bg-zinc-900 border border-zinc-800 rounded-sm shadow-2xl w-full max-w-sm p-8 animate-in zoom-in-50 duration-200">
        <div className="flex flex-col items-center text-center space-y-6">
          <div className="relative">
            <div className="w-16 h-16 bg-red-500/10 rounded-full flex items-center justify-center animate-pulse">
              <Trash2 className="w-8 h-8 text-red-500" />
            </div>
            <div className="absolute inset-0 border-4 border-red-500/20 border-t-red-500 rounded-full animate-spin"></div>
          </div>
          <div>
            <h3 className="text-xl font-bold text-white mb-2 uppercase tracking-wide">Deleting Tunnels</h3>
            <p className="text-zinc-500 text-xs uppercase tracking-widest">Please wait while cleaning up...</p>
          </div>
          <div className="w-full space-y-2">
            <div className="flex justify-between text-[10px] uppercase font-bold text-zinc-500 tracking-wider">
              <span>Progress</span>
              <span>{percentage}%</span>
            </div>
            <div className="w-full h-2 bg-zinc-800 rounded-full overflow-hidden">
              <div
                className="h-full bg-red-600 transition-all duration-300 ease-out"
                style={{ width: `${percentage}%` }}
              ></div>
            </div>
            <p className="text-[10px] text-zinc-600 text-center pt-2">
              STOPPED {progress} OF {total} TUNNELS
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

const ConfirmModal = ({ isOpen, onClose, onConfirm, title, message }) => {
  if (!isOpen) return null;
  return (
    <div className="fixed inset-0 z-[60] flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm transition-all animate-in fade-in duration-200">
      <div className="bg-zinc-900 border border-zinc-800 rounded-sm shadow-2xl w-full max-w-sm p-6 animate-in zoom-in-95 duration-200">
        <h3 className="text-lg font-bold text-white mb-2 uppercase tracking-wide">{title}</h3>
        <p className="text-zinc-400 text-sm mb-6">{message}</p>
        <div className="flex justify-end gap-3">
          <button
            onClick={onClose}
            className="px-4 py-2 text-xs font-bold uppercase text-zinc-500 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={() => { onConfirm(); onClose(); }}
            className="px-4 py-2 text-xs font-bold uppercase bg-red-600 text-white hover:bg-red-700 transition-colors rounded-sm"
          >
            Confirm
          </button>
        </div>
      </div>
    </div>
  );
};

function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [password, setPassword] = useState('');
  const [status, setStatus] = useState({ connected: false, tunnels: {}, domains: {} });
  const [activeTab, setActiveTab] = useState('tunnels');
  const [newTunnel, setNewTunnel] = useState({ public_port: '', local_port: '', protocol: 'TCP' });
  const [newDomain, setNewDomain] = useState({ domain: '', target_port: '', mode: 'auto' });
  const [data, setData] = useState([]);
  const [lastStats, setLastStats] = useState({ up: 0, down: 0, time: Date.now() });
  const [toasts, setToasts] = useState([]);
  const [bulkDeleting, setBulkDeleting] = useState(false);
  const [deleteProgress, setDeleteProgress] = useState({ current: 0, total: 0 });
  const [inspectorLogs, setInspectorLogs] = useState([]);
  const [selectedLogId, setSelectedLogId] = useState(null);
  const [confirmModal, setConfirmModal] = useState({ isOpen: false, title: '', message: '', onConfirm: null });


  const [selectedTunnels, setSelectedTunnels] = useState(new Set());

  useEffect(() => {

    fetchStatus();
    const interval = setInterval(fetchStatus, 1000);
    return () => clearInterval(interval);
  }, [isAuthenticated]);

  const addToast = (message, type = 'error') => {
    const id = Date.now();
    setToasts(prev => [...prev, { id, message, type }]);
  };

  const removeToast = (id) => {
    setToasts(prev => prev.filter(t => t.id !== id));
  };

  const login = async (e) => {
    e.preventDefault();
    try {
      const res = await fetch(`${API_BASE}/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password })
      });
      if (res.ok) {
        setIsAuthenticated(true);
        setPassword('');
      } else {
        addToast("Invalid Password", "error");
      }
    } catch (err) {
      addToast("Login Failed", "error");
    }
  };


  const logout = async () => {
    try {
      await fetch(`${API_BASE}/logout`);
    } catch (e) {

    }
    setIsAuthenticated(false);
    setPassword('');
    setStatus({ connected: false, tunnels: {}, domains: {} });
  };

  const fetchStatus = async () => {
    try {
      const res = await fetch(`${API_BASE}/status`);
      if (res.status === 401) {
        setIsAuthenticated(false);
        setStatus(prev => ({ ...prev, connected: false }));
        return;
      }
      setIsAuthenticated(true);

      const payload = await res.json();
      const now = Date.now();

      setStatus({
        connected: payload.connected,
        tunnels: payload.tunnels,
        domains: payload.domains || {},
        uptime: payload.uptime,
        server_addr: payload.server_addr,
        latency: payload.stats ? payload.stats.latency_ms : 0
      });

      if (payload.stats) {
        setLastStats(prev => {
          const timeDiff = (now - prev.time) / 1000;
          if (timeDiff <= 0) return prev;
          const speedUp = Math.max(0, (payload.stats.bytes_up - prev.up) / timeDiff);
          const speedDown = Math.max(0, (payload.stats.bytes_down - prev.down) / timeDiff);

          setData(curr => {

            const newData = [...curr, { time: now, up: speedUp, down: speedDown }];
            return newData.slice(-60);
          });
          return { up: payload.stats.bytes_up, down: payload.stats.bytes_down, time: now };
        });
      }

      if (activeTab === 'inspector') {
        const iRes = await fetch(`${API_BASE}/inspect`);
        if (iRes.ok) {
          const logs = await iRes.json();
          setInspectorLogs(logs.reverse());
        }
      }
    } catch (e) {


    }
  };

  const addTunnel = async (e) => {
    e.preventDefault();
    if (!newTunnel.public_port || !newTunnel.local_port) {
      addToast("Please fill in all fields", "error");
      return;
    }

    const pub = parseInt(newTunnel.public_port, 10);
    const loc = parseInt(newTunnel.local_port, 10);

    if (!newTunnel.public_port.includes('-') && (pub < 1 || pub > 65535)) {
      addToast("Public port must be 1-65535", "error");
      return;
    }
    if (!newTunnel.local_port.includes('-') && (loc < 1 || loc > 65535)) {
      addToast("Local port must be 1-65535", "error");
      return;
    }

    try {
      const res = await fetch(`${API_BASE}/tunnels`, {
        method: 'POST',
        body: JSON.stringify({
          public_port: newTunnel.public_port.toString(),
          local_port: newTunnel.local_port.toString(),
          protocol: newTunnel.protocol
        })
      });

      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText);
      }

      setNewTunnel(prev => ({ ...prev, public_port: '', local_port: '' }));
      fetchStatus();
      addToast("Tunnel initialized successfully", "success");
    } catch (err) {
      addToast(err.message.replace(/\n/g, ''), "error");
    }
  };

  const deleteTunnel = (publicPort) => {
    setConfirmModal({
      isOpen: true,
      title: 'Stop Tunnel?',
      message: `This will close the tunnel on port :${publicPort}. Are you sure?`,
      onConfirm: async () => {
        try {
          await fetch(`${API_BASE}/tunnels`, {
            method: 'DELETE',
            body: JSON.stringify({ public_port: int(publicPort) })
          });
          fetchStatus();
          if (selectedTunnels.has(publicPort.toString())) {
            const next = new Set(selectedTunnels);
            next.delete(publicPort.toString());
            setSelectedTunnels(next);
          }
          addToast(`Tunnel :${publicPort} closed`, "success");
        } catch (err) {
          addToast("Failed to close tunnel", "error");
        }
      }
    });
  };

  const toggleSelect = (pub) => {
    const next = new Set(selectedTunnels);
    if (next.has(pub)) next.delete(pub);
    else next.add(pub);
    setSelectedTunnels(next);
  }

  const toggleSelectAll = () => {
    const allPubs = Object.keys(status.tunnels || {});
    if (selectedTunnels.size === allPubs.length) {
      setSelectedTunnels(new Set());
    } else {
      setSelectedTunnels(new Set(allPubs));
    }
  }

  const deleteSelected = () => {
    const total = selectedTunnels.size;

    setConfirmModal({
      isOpen: true,
      title: 'Stop Selected Tunnels?',
      message: `You are about to stop ${total} tunnels. This action cannot be undone.`,
      onConfirm: async () => {
        setBulkDeleting(true);
        setDeleteProgress({ current: 0, total });

        const allPorts = Array.from(selectedTunnels).map(p => parseInt(p, 10));
        const CHUNK_SIZE = 50;
        let processed = 0;

        try {
          for (let i = 0; i < allPorts.length; i += CHUNK_SIZE) {
            const chunk = allPorts.slice(i, i + CHUNK_SIZE);

            await fetch(`${API_BASE}/tunnels`, {
              method: 'DELETE',
              headers: { 'Content-Type': 'application/json' },
              body: JSON.stringify({ public_ports: chunk })
            });

            processed += chunk.length;
            setDeleteProgress({ current: processed, total });
            await new Promise(r => setTimeout(r, 50));
          }

          addToast(`STOPPED ${total} TUNNELS`, "success");
          setSelectedTunnels(new Set());
          fetchStatus();
        } catch (err) {
          console.error(err);
          addToast("FAILED TO STOP SOME TUNNELS", "error");
        } finally {
          setTimeout(() => {
            setBulkDeleting(false);
          }, 500);
        }
      }
    });
  }

  const addDomain = async (e) => {
    e.preventDefault();
    if (!newDomain.domain || !newDomain.target_port) {
      addToast("Please fill in all fields", "error");
      return;
    }
    const port = parseInt(newDomain.target_port, 10);
    try {
      const res = await fetch(`${API_BASE}/domains`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          domain: newDomain.domain,
          public_port: port,
          mode: newDomain.mode
        })
      });
      if (!res.ok) throw new Error(await res.text());
      setNewDomain({ domain: '', target_port: '', mode: 'auto' });
      fetchStatus();
      addToast(`Mapped ${newDomain.domain}`, "success");
    } catch (err) {
      addToast("Failed to map domain", "error");
    }
  };

  const deleteDomain = (domain) => {
    setConfirmModal({
      isOpen: true,
      title: 'Unmap Domain?',
      message: `This will remove the mapping for ${domain}. It will no longer point to your tunnel.`,
      onConfirm: async () => {
        try {
          await fetch(`${API_BASE}/domains`, {
            method: 'DELETE',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ domain })
          });
          fetchStatus();
          addToast(`Unmapped ${domain}`, "success");
        } catch (err) {
          addToast("Failed to unmap domain", "error");
        }
      }
    });
  };

  const int = (s) => parseInt(s, 10);
  const tunnelsArr = Object.entries(status.tunnels || {});
  const domainsArr = Object.entries(status.domains || {});





  if (!isAuthenticated) {
    return (
      <div className="min-h-screen bg-black text-zinc-300 font-mono flex items-center justify-center p-4">
        <ToastContainer toasts={toasts} removeToast={removeToast} />
        <div className="w-full max-w-sm border border-zinc-900 bg-zinc-950/50 p-8 rounded-sm shadow-2xl">
          <div className="flex justify-center mb-6">
            <div className="w-12 h-12 bg-white text-black flex items-center justify-center font-bold text-xl rounded-sm">T</div>
          </div>
          <h2 className="text-center text-xl font-bold text-white mb-8 tracking-tight">TunnelCow Access</h2>
          <form onSubmit={login} className="space-y-4">
            <input
              type="password"
              className="w-full bg-black border border-zinc-800 p-3 text-white placeholder-zinc-700 focus:outline-none focus:border-white transition-colors font-mono text-center rounded-sm"
              placeholder="Enter Dashboard Password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              autoFocus
            />
            <button type="submit" className="w-full bg-white text-black font-bold uppercase py-3 hover:bg-zinc-200 transition-colors rounded-sm flex items-center justify-center gap-2">
              <LockOpen className="w-4 h-4" /> Unlock
            </button>
          </form>
          <p className="text-center text-zinc-600 text-xs mt-6 uppercase tracking-widest">Protected System</p>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-black text-zinc-300 font-mono p-4 md:p-8 relative selection:bg-white selection:text-black">
      <ToastContainer toasts={toasts} removeToast={removeToast} />
      <BulkDeleteModal isOpen={bulkDeleting} progress={deleteProgress.current} total={deleteProgress.total} />
      <ConfirmModal
        isOpen={confirmModal.isOpen}
        onClose={() => setConfirmModal(prev => ({ ...prev, isOpen: false }))}
        title={confirmModal.title}
        message={confirmModal.message}
        onConfirm={confirmModal.onConfirm}
      />

      <div className="max-w-6xl mx-auto space-y-6">

        {/* Header */}
        <header className="flex flex-col md:flex-row justify-between items-start md:items-center gap-6 border-b border-zinc-900 pb-6">
          <div className="flex items-center gap-4">
            <img src="/tunnelcow-ico.svg" alt="TunnelCow Logo" className="w-10 h-10 rounded-sm shadow-[0_0_15px_rgba(255,255,255,0.3)] bg-white p-1" />
            <div>
              <h1 className="text-2xl font-bold tracking-tight text-white mb-1">TunnelCow</h1>
              <p className="text-xs text-zinc-500 uppercase tracking-widest flex items-center gap-2">
                Yet Another Tunneling Manager
                <span className="w-1 h-1 bg-zinc-700 rounded-full"></span>
                v0.1.8
              </p>
            </div>
          </div>

          <div className="flex items-center gap-4">
            <div className="text-right">
              <div className="text-[10px] uppercase font-bold text-zinc-600 mb-1">Status</div>
              <div className={clsx(
                "flex items-center gap-2 px-3 py-1.5 rounded-sm border text-xs font-bold uppercase tracking-wider transition-all shadow-[0_0_10px_rgba(0,0,0,0.5)]",
                status.connected
                  ? "bg-zinc-950 border-zinc-800 text-green-500 shadow-[0_0_10px_rgba(34,197,94,0.1)]"
                  : "bg-black border-red-900 text-red-600"
              )}>
                <div className={clsx("w-2 h-2 rounded-full animate-pulse", status.connected ? "bg-green-500" : "bg-red-600")} />
                {status.connected ? `Online (${status.server_addr || '...'})` : 'Disconnected'}
              </div>
            </div>
            <button
              onClick={logout}
              className="w-8 h-8 flex items-center justify-center bg-zinc-900 hover:bg-red-900/30 text-zinc-500 hover:text-red-500 rounded-sm transition-colors"
              title="Logout"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </header>

        {/* Stats */}
        <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
          <StatCard title="Active Tunnels" value={tunnelsArr.length} icon={<Zap className="w-4 h-4 text-yellow-500" />} />
          <StatCard title="Uptime" value={formatTime(status.uptime || 0)} icon={<Activity className="w-4 h-4 text-blue-500" />} />
          <StatCard title="Latency" value={(status.latency || 0) + ' ms'} icon={<Globe className="w-4 h-4 text-purple-500" />} />
          <StatCard title="Upload" value={formatBytes(lastStats.up) + '/s'} icon={<ArrowUpRight className="w-4 h-4 text-orange-500" />} />
          <StatCard title="Download" value={formatBytes(lastStats.down) + '/s'} icon={<ArrowUpRight className="w-4 h-4 text-cyan-500 rotate-180" />} />
        </div>

        {/* Traffic Chart */}
        <div className="border border-zinc-900 bg-zinc-950/30 p-4 rounded-sm relative overflow-hidden group">
          <div className="absolute inset-0 bg-gradient-to-b from-transparent via-transparent to-zinc-900/10 pointer-events-none" />
          <div className="flex justify-between items-center mb-4 relative z-10">
            <h3 className="text-xs font-bold text-zinc-500 uppercase flex items-center gap-2">
              <Activity className="w-4 h-4" /> Live Traffic
            </h3>
            <div className="flex gap-4 text-xs font-mono">
              <span className="text-white flex items-center gap-1"><div className="w-2 h-2 bg-white rounded-full"></div> Upload</span>
              <span className="text-zinc-500 flex items-center gap-1"><div className="w-2 h-2 bg-zinc-600 rounded-full"></div> Download</span>
            </div>
          </div>
          <div className="h-64 w-full relative z-10">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={data}>
                <defs>
                  <linearGradient id="gradUp" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#fff" stopOpacity={0.2} />
                    <stop offset="95%" stopColor="#fff" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="gradDown" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#52525b" stopOpacity={0.2} />
                    <stop offset="95%" stopColor="#52525b" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <Tooltip
                  contentStyle={{ backgroundColor: '#09090b', borderColor: '#27272a', color: '#fff', fontSize: '12px' }}
                  itemStyle={{ color: '#fff' }}
                  labelStyle={{ display: 'none' }}
                  formatter={(value) => [formatBytes(value) + '/s', 'Speed']}
                />
                <Area type="monotone" dataKey="up" stroke="#fff" fill="url(#gradUp)" strokeWidth={2} isAnimationActive={false} />
                <Area type="monotone" dataKey="down" stroke="#52525b" fill="url(#gradDown)" strokeWidth={2} isAnimationActive={false} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </div>

        {/* Tabs */}
        <div className="flex gap-4 border-b border-zinc-900">
          <button
            onClick={() => setActiveTab('tunnels')}
            className={clsx("pb-2 px-1 text-sm font-bold uppercase tracking-wider transition-colors flex items-center gap-2", activeTab === 'tunnels' ? "text-white border-b-2 border-white" : "text-zinc-600 hover:text-zinc-400")}
          >
            <Zap className="w-4 h-4" /> Tunnels
          </button>
          <button
            onClick={() => setActiveTab('domains')}
            className={clsx("pb-2 px-1 text-sm font-bold uppercase tracking-wider transition-colors flex items-center gap-2", activeTab === 'domains' ? "text-white border-b-2 border-white" : "text-zinc-600 hover:text-zinc-400")}
          >
            <Globe className="w-4 h-4" /> Domains
          </button>
          <button
            onClick={() => setActiveTab('inspector')}
            className={clsx("pb-2 px-1 text-sm font-bold uppercase tracking-wider transition-colors flex items-center gap-2", activeTab === 'inspector' ? "text-white border-b-2 border-white" : "text-zinc-600 hover:text-zinc-400")}
          >
            <Eye className="w-4 h-4" /> Inspector
          </button>
        </div>

        {activeTab === 'tunnels' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-in fade-in duration-300">
            {/* List */}
            <div className="lg:col-span-2 border border-zinc-900 bg-zinc-950/30 rounded-sm flex flex-col h-[500px]">
              <div className="px-6 py-4 border-b border-zinc-900 flex justify-between items-center bg-zinc-950/50">
                <div className="flex items-center gap-3">
                  <button onClick={toggleSelectAll} className="text-zinc-500 hover:text-white transition-colors">
                    {selectedTunnels.size > 0 && selectedTunnels.size === tunnelsArr.length ? <CheckSquare className="w-4 h-4" /> : <Square className="w-4 h-4" />}
                  </button>
                  <h3 className="text-xs font-bold text-zinc-500 uppercase">
                    {selectedTunnels.size > 0 ? `${selectedTunnels.size} Selected` : 'Active Tunnels'}
                  </h3>
                </div>
                {selectedTunnels.size > 0 && (
                  <button
                    onClick={deleteSelected}
                    className="text-[10px] bg-red-900/20 text-red-500 border border-red-900/50 px-3 py-1 rounded-sm uppercase font-bold hover:bg-red-900/40 transition-colors flex items-center gap-2"
                  >
                    <Trash2 className="w-3 h-3" /> Stop Selected
                  </button>
                )}
              </div>
              <div className="divide-y divide-zinc-900 overflow-y-auto custom-scrollbar flex-1">
                {tunnelsArr.length === 0 ? (
                  <div className="h-full flex flex-col items-center justify-center text-zinc-800 gap-2">
                    <Server className="w-12 h-12 opacity-20" />
                    <span className="text-sm font-bold opacity-50">No active tunnels</span>
                  </div>
                ) : (
                  tunnelsArr.map(([pub, local]) => (
                    <div key={pub} className={clsx(
                      "p-4 flex items-center justify-between transition-all group border-l-2",
                      selectedTunnels.has(pub) ? "bg-zinc-900/50 border-white" : "hover:bg-zinc-900/20 border-transparent"
                    )}>
                      <div className="flex items-center gap-4">
                        <button onClick={() => toggleSelect(pub)} className={clsx("transition-colors", selectedTunnels.has(pub) ? "text-white" : "text-zinc-700 hover:text-zinc-500")}>
                          {selectedTunnels.has(pub) ? <CheckSquare className="w-4 h-4" /> : <Square className="w-4 h-4" />}
                        </button>
                        <div className="flex items-center gap-6">
                          <div className="flex flex-col">
                            <span className="text-[10px] text-zinc-600 font-bold uppercase">Public</span>
                            <span className="text-lg font-mono text-white">:{pub}</span>
                          </div>
                          <ArrowUpRight className="text-zinc-800 w-4 h-4" />
                          <div className="flex flex-col">
                            <span className="text-[10px] text-zinc-600 font-bold uppercase">Local</span>
                            <span className="text-lg font-mono text-white">:{local}</span>
                          </div>
                        </div>
                      </div>
                      <button
                        onClick={() => deleteTunnel(pub)}
                        className="opacity-0 group-hover:opacity-100 p-2 text-zinc-600 hover:text-red-500 hover:bg-red-950/30 rounded-full transition-all"
                        title="Stop Tunnel"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Config Form */}
            <div className="border border-zinc-900 bg-zinc-950/30 rounded-sm p-6 h-fit sticky top-6">
              <h3 className="text-xs font-bold text-zinc-500 uppercase mb-4 flex items-center gap-2">
                <Plus className="w-4 h-4" /> Initialize Tunnel
              </h3>
              <form onSubmit={addTunnel} className="space-y-4">
                <div>
                  <label className="block text-[10px] uppercase text-zinc-600 font-bold mb-1">Local Port [Single or Range]</label>
                  <input
                    type="text"
                    className="w-full bg-black border border-zinc-800 p-3 text-white placeholder-zinc-800 focus:outline-none focus:border-white transition-colors font-mono text-sm rounded-sm"
                    placeholder="e.g. 3000"
                    value={newTunnel.local_port}
                    onChange={e => setNewTunnel({ ...newTunnel, local_port: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-[10px] uppercase text-zinc-600 font-bold mb-1">Public Port [Single or Range]</label>
                  <input
                    type="text"
                    className="w-full bg-black border border-zinc-800 p-3 text-white placeholder-zinc-800 focus:outline-none focus:border-white transition-colors font-mono text-sm rounded-sm"
                    placeholder="e.g. 80"
                    value={newTunnel.public_port}
                    onChange={e => setNewTunnel({ ...newTunnel, public_port: e.target.value })}
                  />
                </div>
                <div>
                  <label className="block text-[10px] uppercase text-zinc-600 font-bold mb-1">Protocol</label>
                  <select
                    className="w-full bg-black border border-zinc-800 p-3 text-white focus:outline-none focus:border-white transition-colors font-mono text-sm rounded-sm"
                    value={newTunnel.protocol}
                    onChange={e => setNewTunnel({ ...newTunnel, protocol: e.target.value })}
                  >
                    <option value="TCP">TCP</option>
                    <option value="UDP">UDP</option>
                  </select>
                </div>
                <button type="submit" className="w-full bg-white text-black font-bold text-sm uppercase py-3 hover:bg-zinc-200 transition-colors flex items-center justify-center gap-2 mt-2 rounded-sm active:scale-95 transform duration-100">
                  <Plus className="w-4 h-4" /> Start Tunnel
                </button>
              </form>
            </div>
          </div>
        )}

        {activeTab === 'domains' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-in fade-in duration-300">
            {/* List */}
            <div className="lg:col-span-2 border border-zinc-900 bg-zinc-950/30 rounded-sm flex flex-col h-[500px]">
              <div className="px-6 py-4 border-b border-zinc-900 flex justify-between items-center bg-zinc-950/50">
                <h3 className="text-xs font-bold text-zinc-500 uppercase">
                  Active Domains
                </h3>
              </div>
              <div className="divide-y divide-zinc-900 overflow-y-auto custom-scrollbar flex-1">
                {domainsArr.length === 0 ? (
                  <div className="h-full flex flex-col items-center justify-center text-zinc-800 gap-2">
                    <Globe className="w-12 h-12 opacity-20" />
                    <span className="text-sm font-bold opacity-50">No mapped domains</span>
                  </div>
                ) : (
                  domainsArr.map(([domain, port]) => (
                    <div key={domain} className="p-4 flex items-center justify-between hover:bg-zinc-900/20 transition-all border-l-2 border-transparent hover:border-zinc-800">
                      <div className="flex items-center gap-6">
                        <div className="flex flex-col">
                          <span className="text-[10px] text-zinc-600 font-bold uppercase">Domain</span>
                          <span className="text-lg font-mono text-white flex items-center gap-2">
                            {domain} <Shield className="w-3 h-3 text-green-500" />
                          </span>
                        </div>
                        <ArrowUpRight className="text-zinc-800 w-4 h-4" />
                        <div className="flex flex-col">
                          <span className="text-[10px] text-zinc-600 font-bold uppercase">Target</span>
                          <span className="text-lg font-mono text-white flex items-center gap-2">
                            :{port && typeof port === 'object' ? port.public_port : port}
                            {port && typeof port === 'object' && (
                              <span className={clsx(
                                "text-[10px] px-1.5 py-0.5 rounded border uppercase font-bold",
                                port.mode === 'http' ? "border-zinc-700 text-zinc-500" : "border-green-900 text-green-500 bg-green-950/20"
                              )}>
                                {port.mode || 'AUTO'}
                              </span>
                            )}
                          </span>
                        </div>
                      </div>
                      <button
                        onClick={() => deleteDomain(domain)}
                        className="p-2 text-zinc-600 hover:text-red-500 hover:bg-red-950/30 rounded-full transition-all"
                        title="Unmap Domain"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Config Form */}
            <div className="border border-zinc-900 bg-zinc-950/30 rounded-sm p-6 h-fit sticky top-6">
              <h3 className="text-xs font-bold text-zinc-500 uppercase mb-4 flex items-center gap-2">
                <Plus className="w-4 h-4" /> New Domain Map
              </h3>
              <form onSubmit={addDomain} className="space-y-4">
                <div>
                  <label className="block text-[10px] uppercase text-zinc-600 font-bold mb-1">Domain Name</label>
                  <input
                    type="text"
                    className="w-full bg-black border border-zinc-800 p-3 text-white placeholder-zinc-800 focus:outline-none focus:border-white transition-colors font-mono text-sm rounded-sm"
                    placeholder="e.g. app.myserver.com"
                    value={newDomain.domain}
                    onChange={e => setNewDomain({ ...newDomain, domain: e.target.value })}
                  />
                  <p className="text-[10px] text-zinc-600 mt-1">
                    * Make sure A Record points to {status.server_addr?.split(':')[0] || 'Server IP'}
                  </p>
                </div>
                <div>
                  <label className="block text-[10px] uppercase text-zinc-600 font-bold mb-1">Target Tunnel (Public Port)</label>
                  <select
                    className="w-full bg-black border border-zinc-800 p-3 text-white focus:outline-none focus:border-white transition-colors font-mono text-sm rounded-sm"
                    value={newDomain.target_port}
                    onChange={e => setNewDomain({ ...newDomain, target_port: e.target.value })}
                  >
                    <option value="">-- Select Tunnel --</option>
                    {tunnelsArr.map(([pub, local]) => (
                      <option key={pub} value={pub}>
                        :{pub} (Local :{local})
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-[10px] uppercase text-zinc-600 font-bold mb-1">SSL Mode</label>
                  <select
                    className="w-full bg-black border border-zinc-800 p-3 text-white focus:outline-none focus:border-white transition-colors font-mono text-sm rounded-sm"
                    value={newDomain.mode}
                    onChange={e => setNewDomain({ ...newDomain, mode: e.target.value })}
                  >
                    <option value="auto">Auto HTTPS (Standard)</option>
                    <option value="http">HTTP Only (Insecure)</option>
                    <option value="https">HTTPS Only (Strict)</option>
                  </select>
                </div>

                <button type="submit" className="w-full bg-white text-black font-bold text-sm uppercase py-3 hover:bg-zinc-200 transition-colors flex items-center justify-center gap-2 mt-2 rounded-sm active:scale-95 transform duration-100">
                  <Shield className="w-4 h-4" /> Secure & Map
                </button>
              </form>
            </div>
          </div>
        )}

        {activeTab === 'inspector' && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 animate-in fade-in duration-300 h-[600px]">
            {/* Request List */}
            <div className="border border-zinc-900 bg-zinc-950/30 rounded-sm flex flex-col overflow-hidden">
              <div className="px-4 py-3 border-b border-zinc-900 bg-zinc-950/50 flex justify-between items-center">
                <h3 className="text-xs font-bold text-zinc-500 uppercase flex items-center gap-2">
                  <Activity className="w-4 h-4" /> Live Requests
                </h3>
                <span className="text-[10px] bg-zinc-900 text-zinc-500 px-2 py-1 rounded-full">{inspectorLogs.length} events</span>
              </div>
              <div className="flex-1 overflow-y-auto custom-scrollbar divide-y divide-zinc-900">
                {inspectorLogs.length === 0 ? (
                  <div className="h-full flex flex-col items-center justify-center text-zinc-800 gap-2 p-8">
                    <Search className="w-8 h-8 opacity-20" />
                    <span className="text-xs font-bold opacity-50">Waiting for traffic...</span>
                  </div>
                ) : (
                  inspectorLogs.map(log => (
                    <div
                      key={log.id}
                      onClick={() => setSelectedLogId(log.id)}
                      className={clsx(
                        "p-3 cursor-pointer hover:bg-zinc-900/40 transition-colors border-l-2 flex flex-col gap-1",
                        selectedLogId === log.id ? "bg-zinc-900/60 border-blue-500" : "border-transparent"
                      )}
                    >
                      <div className="flex justify-between items-start">
                        <span className={clsx("text-[10px] font-bold px-1.5 py-0.5 rounded",
                          log.method === 'GET' ? 'bg-blue-900/20 text-blue-500' :
                            log.method === 'POST' ? 'bg-green-900/20 text-green-500' :
                              log.method === 'DELETE' ? 'bg-red-900/20 text-red-500' : 'bg-zinc-800 text-zinc-400'
                        )}>{log.method}</span>
                        <span className={clsx("text-[10px] font-mono", log.status >= 400 ? "text-red-500" : "text-green-500")}>
                          {log.status}
                        </span>
                      </div>
                      <div className="text-xs text-zinc-300 truncate font-mono" title={log.url}>{log.url}</div>
                      <div className="flex justify-between items-center mt-1">
                        <span className="text-[10px] text-zinc-600 flex items-center gap-1">
                          <Clock className="w-3 h-3" /> {log.duration_ms}ms
                        </span>
                        <span className="text-[10px] text-zinc-700">{new Date(log.timestamp).toLocaleTimeString()}</span>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </div>

            {/* Details Panel */}
            <div className="lg:col-span-2 border border-zinc-900 bg-zinc-950/30 rounded-sm flex flex-col overflow-hidden relative">
              {inspectorLogs.find(l => l.id === selectedLogId) ? (
                (() => {
                  const log = inspectorLogs.find(l => l.id === selectedLogId);
                  return (
                    <div className="flex flex-col h-full">
                      <div className="px-6 py-4 border-b border-zinc-900 bg-zinc-950/50 flex justify-between items-start">
                        <div className="flex flex-col gap-1">
                          <h2 className="text-lg font-bold text-white font-mono break-all">{log.method} {log.url}</h2>
                          <div className="flex gap-4 text-xs text-zinc-500">
                            <span>ID: {log.id}</span>
                            <span>IP: {log.client_ip}</span>
                            <span>Time: {new Date(log.timestamp).toLocaleString()}</span>
                          </div>
                        </div>
                        <div className={clsx("text-xl font-bold px-3 py-1 rounded border",
                          log.status >= 400 ? "border-red-900/50 text-red-500 bg-red-950/10" : "border-green-900/50 text-green-500 bg-green-950/10"
                        )}>
                          {log.status}
                        </div>
                      </div>

                      <div className="flex-1 overflow-y-auto custom-scrollbar p-6 space-y-8">
                        {/* Request */}
                        <div>
                          <h3 className="text-xs font-bold text-zinc-500 uppercase mb-3 flex items-center gap-2">
                            <ArrowUpRight className="w-4 h-4" /> Request Headers
                          </h3>
                          <div className="bg-black/50 border border-zinc-900 rounded p-4 font-mono text-xs text-zinc-400 overflow-x-auto">
                            {Object.entries(log.req_headers || {}).map(([k, v]) => (
                              <div key={k} className="flex gap-2">
                                <span className="text-blue-400 font-bold min-w-[120px]">{k}:</span>
                                <span className="break-all">{v}</span>
                              </div>
                            ))}
                          </div>
                        </div>

                        {log.req_body && (
                          <div>
                            <h3 className="text-xs font-bold text-zinc-500 uppercase mb-3 flex items-center gap-2">
                              <Code className="w-4 h-4" /> Request Body
                            </h3>
                            <pre className="bg-black/50 border border-zinc-900 rounded p-4 font-mono text-xs text-zinc-300 overflow-x-auto whitespace-pre-wrap">{log.req_body}</pre>
                          </div>
                        )}

                        {/* Response */}
                        <div>
                          <h3 className="text-xs font-bold text-zinc-500 uppercase mb-3 flex items-center gap-2">
                            <ArrowUpRight className="w-4 h-4 rotate-180" /> Response Headers
                          </h3>
                          <div className="bg-black/50 border border-zinc-900 rounded p-4 font-mono text-xs text-zinc-400 overflow-x-auto">
                            {Object.entries(log.res_headers || {}).map(([k, v]) => (
                              <div key={k} className="flex gap-2">
                                <span className="text-purple-400 font-bold min-w-[120px]">{k}:</span>
                                <span className="break-all">{v}</span>
                              </div>
                            ))}
                          </div>
                        </div>

                        {log.res_body && (
                          <div>
                            <h3 className="text-xs font-bold text-zinc-500 uppercase mb-3 flex items-center gap-2">
                              <Code className="w-4 h-4" /> Response Body
                            </h3>
                            <pre className="bg-black/50 border border-zinc-900 rounded p-4 font-mono text-xs text-zinc-300 overflow-x-auto whitespace-pre-wrap">
                              {log.res_body.length > 2000 ? log.res_body.substring(0, 2000) + '... (Truncated)' : log.res_body}
                            </pre>
                          </div>
                        )}

                      </div>
                    </div>
                  );
                })()
              ) : (
                <div className="h-full flex flex-col items-center justify-center text-zinc-700">
                  <div className="w-16 h-16 border-2 border-zinc-800 rounded-lg flex items-center justify-center mb-4">
                    <Activity className="w-8 h-8" />
                  </div>
                  <p className="text-sm font-bold uppercase tracking-wider">Select a request to view details</p>
                </div>
              )}
            </div>
          </div>
        )}

      </div>
    </div>
  );
}

function ToastContainer({ toasts, removeToast }) {
  return createPortal(
    <div className="fixed bottom-6 right-6 flex flex-col gap-2 z-50 pointer-events-none">
      {toasts.map(toast => (
        <Toast key={toast.id} toast={toast} onRemove={removeToast} />
      ))}
    </div>,
    document.body
  );
}

function StatCard({ title, value, icon }) {
  return (
    <div className="border border-zinc-900 bg-zinc-950/30 p-4 flex flex-col gap-2 hover:border-zinc-800 transition-colors group">
      <div className="flex justify-between items-start">
        <h4 className="text-[10px] font-bold text-zinc-600 uppercase tracking-wider">{title}</h4>
        <div className="opacity-50 group-hover:opacity-100 transition-opacity">{icon}</div>
      </div>
      <div className="text-xl font-mono text-white">{value}</div>
    </div>
  )
}

function formatBytes(bytes, decimals = 2) {
  if (!+bytes) return '0 B';
  const k = 1024;
  const dm = decimals < 0 ? 0 : decimals;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(dm))} ${sizes[i]}`;
}

function formatTime(seconds) {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
}


function Toast({ toast, onRemove }) {
  const [mounted, setMounted] = useState(false);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    setMounted(true);
    const enterTimer = setTimeout(() => setVisible(true), 10);
    const autoCloseTimer = setTimeout(() => handleClose(), 5000);
    return () => { clearTimeout(enterTimer); clearTimeout(autoCloseTimer); };
  }, []);

  const handleClose = () => {
    setVisible(false);
    setTimeout(() => onRemove(toast.id), 400);
  };

  if (!mounted) return null;

  return (
    <div
      className={clsx(
        "pointer-events-auto flex items-center gap-3 p-4 rounded-sm border shadow-2xl w-[350px] backdrop-blur-md relative overflow-hidden group",
        toast.type === 'error' ? "bg-black/90 border-red-900 text-red-500" : "bg-black/90 border-zinc-800 text-white"
      )}
      style={{
        transition: 'all 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
        transform: visible ? 'translateX(0)' : 'translateX(100%)',
        opacity: visible ? 1 : 0,
      }}
    >
      <div className={clsx("absolute left-0 top-0 bottom-0 w-1", toast.type === 'error' ? "bg-red-500" : "bg-green-500")} />

      {toast.type === 'error' ? <AlertCircle className="w-5 h-5 shrink-0" /> : <Shield className="w-5 h-5 shrink-0 text-green-500" />}
      <div className="flex-1 flex flex-col">
        <p className="text-xs font-bold uppercase tracking-wide">{toast.message}</p>
        <p className="text-[10px] text-zinc-500 uppercase tracking-widest mt-0.5">Notification</p>
      </div>
      <button onClick={handleClose} className="text-zinc-600 hover:text-white transition-colors opacity-0 group-hover:opacity-100">
        <X className="w-4 h-4" />
      </button>
    </div>
  );
}

export default App;



























