using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Data;
using System.Diagnostics;
using System.Drawing;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Windows.Forms;

namespace NextDNS
{
    public partial class SettingsForm : Form
    {
        private Service.Client service;
        private bool enabled = false; // Actual status

        public SettingsForm()
        {
            service = new Service.Client();
            service.EventReceived += Service_EventReceived;
            service.Connected += Service_Connected;
            Properties.Settings.Default.SettingsSaving += Default_SettingsSaving;

            InitializeComponent();
            Hide();

            configuration.Text = Properties.Settings.Default.Configuration;
            reportDeviceName.Checked = Properties.Settings.Default.ReportDeviceName;
            checkUpdate.Checked = Properties.Settings.Default.CheckUpdates;
        }

        async private void Service_Connected(object sender, EventArgs e)
        {
            sendSettings();

            try
            {
                if (Properties.Settings.Default.Enabled)
                {
                    // If enabled, auto-connect on start
                    await service.SendAsync(new Service.Event("enable")).ConfigureAwait(false);
                }
                else
                {
                    // Update status
                    await service.SendAsync(new Service.Event("status")).ConfigureAwait(false);
                }
            }
            catch (Exception)
            {
                // Silient failure on startup
            }
        }

        private void Default_SettingsSaving(object sender, CancelEventArgs e)
        {
            sendSettings();
        }

        async private void sendSettings()
        {
            var settings = new Service.Event("settings");
            settings.data = new Service.EventData();
            settings.data.configuration = Properties.Settings.Default.Configuration;
            settings.data.reportDeviceName = Properties.Settings.Default.ReportDeviceName;
            settings.data.checkUpdates = Properties.Settings.Default.CheckUpdates;
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
                    Debug.WriteLine("status {0}", (object)(e.data.enabled ? "enabled" : "disabled"));
                    enabled = e.data.enabled;
                    toggle.Text = enabled ? "Disable" : "Enable";
                    break;
                case "error":
                    MessageBox.Show(e.data.error, "NextDNS Error", MessageBoxButtons.OK, MessageBoxIcon.Error);
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

        async private void toggle_Click(object sender, EventArgs e)
        {
            try
            {
                Properties.Settings.Default.Enabled = !enabled; // Store the intent
                Properties.Settings.Default.Save();
                await service.SendAsync(new Service.Event(enabled ? "disable" : "enable")).ConfigureAwait(false);
            }
            catch (Exception ex)
            {
                Debug.WriteLine("Service broadcast error: {0}", (object)ex.Message);
                MessageBox.Show("An error append while trying to communicate with NextDNS Windows service.",
                    "Service Communication Error", 
                    MessageBoxButtons.OK, MessageBoxIcon.Error);
            }
        }

        async private void quit_Click(object sender, EventArgs e)
        {
            if (enabled)
            {
                try
                {
                    // Disconnect on exit
                    await service.SendAsync(new Service.Event("disable")).ConfigureAwait(false);
                }
                catch (Exception)
                {
                    // Silient failure on exit
                }
            }

            Environment.Exit(1);
        }
    }
}
