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

        public SettingsForm()
        {
            service = new Service.Client();
            service.EventReceived += Service_EventReceived;
            InitializeComponent();
            Hide();
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
                    toggle.Text = e.data.enabled ? "Disable" : "Enable";
                    break;
                case "settings":
                    if (e.data != null)
                    {
                        configuration.Text = e.data.configuration;
                        reportDeviceName.Checked = e.data.reportDeviceName;
                    } 
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

        async private void save_Click(object sender, EventArgs e)
        {
            Hide();

            var settings = new Service.Event("settings");
            settings.data = new Service.EventData();
            settings.data.configuration = configuration.Text;
            settings.data.reportDeviceName = reportDeviceName.Checked;
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
                await service.SendAsync(new Service.Event(toggle.Text.ToLower())).ConfigureAwait(false);
            }
            catch (Exception ex)
            {
                Debug.WriteLine("Service broadcast error: {0}", (object)ex.Message);
                MessageBox.Show("An error append while trying to communicate with NextDNS Windows service.",
                    "Service Communication Error", 
                    MessageBoxButtons.OK, MessageBoxIcon.Error);
            }
        }

        private void quit_Click(object sender, EventArgs e)
        {
            Environment.Exit(1);
        }
    }
}
