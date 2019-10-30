using System;
using System.ComponentModel;
using System.Diagnostics;
using System.Threading.Tasks;
using System.Windows.Forms;

namespace NextDNS
{
    public partial class SettingsForm : Form
    {
        private const string StateStopped = "stopped";
        private const string StateStarting = "starting";
        private const string StateStarted = "started";
        private const string StateReasserting = "reasserting";
        private const string StateStopping = "stopping";

        private Service.Client service;
        private string State = StateStopped; // Last known state

        public SettingsForm()
        {
            InitializeComponent();
            Hide();

            service = new Service.Client();
            service.EventReceived += Service_EventReceived;
            service.Connected += Service_Connected;
            Properties.Settings.Default.SettingsSaving += Default_SettingsSaving;
            service.Connect();

            configuration.Text = Properties.Settings.Default.Configuration;
            reportDeviceName.Checked = Properties.Settings.Default.ReportDeviceName;
            checkUpdate.Checked = Properties.Settings.Default.CheckUpdates;
            updateChannel.SelectedIndex = Properties.Settings.Default.UpdateChannel == "Stable" ? 0 : 1;

            if (Properties.Settings.Default.Configuration.Length == 0)
            {
                Show();
            }
        }

        private void Service_Connected(object sender, EventArgs e)
        {
            sendSettings();
        }

        private void Default_SettingsSaving(object sender, CancelEventArgs e)
        {
            sendSettings();
        }

        async private void sendSettings()
        {
            if (InvokeRequired)
            {
                Invoke((MethodInvoker)delegate { sendSettings(); });
                return;
            }

            var settings = new Service.Event("settings");
            settings.data = new Service.EventData();
            settings.data.enabled = Properties.Settings.Default.Enabled;
            settings.data.configuration = Properties.Settings.Default.Configuration;
            settings.data.reportDeviceName = Properties.Settings.Default.ReportDeviceName;
            settings.data.checkUpdates = Properties.Settings.Default.CheckUpdates;
            settings.data.updateChannel = Properties.Settings.Default.UpdateChannel;
            try
            {
                await service.SendAsync(settings).ConfigureAwait(false);
            }
            catch (Exception ex)
            {
                Debug.WriteLine("Service broadcast error: {0}", (object)ex.Message);
                MessageBox.Show("An error append while trying to communicate with NextDNS Windows service.",
                    "Service Communication Error",
                    MessageBoxButtons.OK, MessageBoxIcon.Error);
            }
        }

        private void Service_EventReceived(object sender, Service.Event e)
        {
            if (InvokeRequired)
            {
                Invoke((MethodInvoker)delegate { Service_EventReceived(sender, e); });
                return;
            }
            switch (e.name)
            {
                case "open":
                    // Called when a new instance of the app is started while an instance is already running.
                    Show();
                    WindowState = FormWindowState.Normal;
                    break;
                case "status":
                    Debug.WriteLine("state {0}", (object)e.data.state);
                    State = e.data.state;
                    switch (State)
                    {
                        case StateStopped:
                            toggle.Text = "Enable";
                            break;
                        case StateStopping:
                            toggle.Text = "Disonnecting...";
                            break;
                        case StateStarting:
                        case StateReasserting:
                            toggle.Text = "Connecting...";
                            break;
                        case StateStarted:
                            toggle.Text = "Disable";
                            break;

                    }
                    status.Text = State;
                    if (e.data.error != null && e.data.error != "")
                    {
                        MessageBox.Show(e.data.error, "NextDNS Error", MessageBoxButtons.OK, MessageBoxIcon.Error);
                    }
                    break;
                default:
                    break;
            }
        }

        private void SettingsForm_FormClosing(object sender, FormClosingEventArgs e)
        {
            if (e.CloseReason != CloseReason.TaskManagerClosing &&
                e.CloseReason != CloseReason.WindowsShutDown)
            {
                e.Cancel = true;
            }
            Hide();
        }

        private void save_Click(object sender, EventArgs e)
        {
            Hide();

            Properties.Settings.Default.Configuration = configuration.Text;
            Properties.Settings.Default.ReportDeviceName = reportDeviceName.Checked;
            Properties.Settings.Default.CheckUpdates = checkUpdate.Checked;
            Properties.Settings.Default.UpdateChannel = updateChannel.SelectedIndex == 0 ? "Stable" : "Beta";

            if (Properties.Settings.Default.Configuration.Length > 0 && !Properties.Settings.Default.Enabled)
            {
                Properties.Settings.Default.Enabled = true;
            }

            Properties.Settings.Default.Save();
        }

        private void cancel_Click(object sender, EventArgs e)
        {
            Hide();
        }

        private void settings_Click(object sender, EventArgs e)
        {
            Show();
            WindowState = FormWindowState.Normal;
        }

        private void toggle_Click(object sender, EventArgs e)
        {
            Properties.Settings.Default.Enabled = State == StateStopped;
            Properties.Settings.Default.Save();
        }

        async private void quit_Click(object sender, EventArgs e)
        {
            if (State != StateStopped)
            {
                try
                {
                    // Disconnect set flush settings on exit
                    var settings = new Service.Event("settings");
                    settings.data = new Service.EventData();
                    settings.data.enabled = false;
                    await service.SendAsync(settings).ConfigureAwait(false);
                }
                catch (Exception)
                {
                    // Silient failure on exit
                }
            }

            Environment.Exit(1);
        }

        private void checkUpdate_CheckedChanged(object sender, EventArgs e)
        {
            updateChannel.Enabled = checkUpdate.Checked;
        }

        private void systray_MouseDoubleClick(object sender, MouseEventArgs e)
        {
            Show();
            WindowState = FormWindowState.Normal;
        }
    }
}
