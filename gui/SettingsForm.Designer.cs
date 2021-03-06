﻿namespace NextDNS
{
    partial class SettingsForm
    {
        /// <summary>
        /// Required designer variable.
        /// </summary>
        private System.ComponentModel.IContainer components = null;

        /// <summary>
        /// Clean up any resources being used.
        /// </summary>
        /// <param name="disposing">true if managed resources should be disposed; otherwise, false.</param>
        protected override void Dispose(bool disposing)
        {
            if (disposing && (components != null))
            {
                components.Dispose();
            }
            base.Dispose(disposing);
        }

        #region Windows Form Designer generated code

        /// <summary>
        /// Required method for Designer support - do not modify
        /// the contents of this method with the code editor.
        /// </summary>
        private void InitializeComponent()
        {
            this.components = new System.ComponentModel.Container();
            System.ComponentModel.ComponentResourceManager resources = new System.ComponentModel.ComponentResourceManager(typeof(SettingsForm));
            this.systray = new System.Windows.Forms.NotifyIcon(this.components);
            this.systrayContextMenu = new System.Windows.Forms.ContextMenuStrip(this.components);
            this.toggle = new System.Windows.Forms.ToolStripMenuItem();
            this.settings = new System.Windows.Forms.ToolStripMenuItem();
            this.toolStripSeparator1 = new System.Windows.Forms.ToolStripSeparator();
            this.quit = new System.Windows.Forms.ToolStripMenuItem();
            this.configurationLabel = new System.Windows.Forms.Label();
            this.configuration = new System.Windows.Forms.TextBox();
            this.generalGroupBox = new System.Windows.Forms.GroupBox();
            this.updateChannelLabel = new System.Windows.Forms.Label();
            this.updateChannel = new System.Windows.Forms.ComboBox();
            this.checkUpdate = new System.Windows.Forms.CheckBox();
            this.reportDeviceName = new System.Windows.Forms.CheckBox();
            this.save = new System.Windows.Forms.Button();
            this.cancel = new System.Windows.Forms.Button();
            this.status = new System.Windows.Forms.Label();
            this.statusGroupBox = new System.Windows.Forms.GroupBox();
            this.systrayContextMenu.SuspendLayout();
            this.generalGroupBox.SuspendLayout();
            this.statusGroupBox.SuspendLayout();
            this.SuspendLayout();
            // 
            // systray
            // 
            this.systray.ContextMenuStrip = this.systrayContextMenu;
            this.systray.Icon = ((System.Drawing.Icon)(resources.GetObject("systray.Icon")));
            this.systray.Text = "NextDNS";
            this.systray.Visible = true;
            this.systray.MouseDoubleClick += new System.Windows.Forms.MouseEventHandler(this.systray_MouseDoubleClick);
            // 
            // systrayContextMenu
            // 
            this.systrayContextMenu.ImageScalingSize = new System.Drawing.Size(32, 32);
            this.systrayContextMenu.Items.AddRange(new System.Windows.Forms.ToolStripItem[] {
            this.toggle,
            this.settings,
            this.toolStripSeparator1,
            this.quit});
            this.systrayContextMenu.Name = "systrayContextMenu";
            this.systrayContextMenu.Size = new System.Drawing.Size(192, 124);
            // 
            // toggle
            // 
            this.toggle.Name = "toggle";
            this.toggle.Size = new System.Drawing.Size(191, 38);
            this.toggle.Text = "Enable";
            this.toggle.Click += new System.EventHandler(this.toggle_Click);
            // 
            // settings
            // 
            this.settings.Name = "settings";
            this.settings.Size = new System.Drawing.Size(191, 38);
            this.settings.Text = "Settings...";
            this.settings.Click += new System.EventHandler(this.settings_Click);
            // 
            // toolStripSeparator1
            // 
            this.toolStripSeparator1.Name = "toolStripSeparator1";
            this.toolStripSeparator1.Size = new System.Drawing.Size(188, 6);
            // 
            // quit
            // 
            this.quit.Name = "quit";
            this.quit.Size = new System.Drawing.Size(191, 38);
            this.quit.Text = "Quit";
            this.quit.Click += new System.EventHandler(this.quit_Click);
            // 
            // configurationLabel
            // 
            this.configurationLabel.AutoSize = true;
            this.configurationLabel.Location = new System.Drawing.Point(6, 62);
            this.configurationLabel.Margin = new System.Windows.Forms.Padding(4, 0, 4, 0);
            this.configurationLabel.Name = "configurationLabel";
            this.configurationLabel.Size = new System.Drawing.Size(172, 25);
            this.configurationLabel.TabIndex = 2;
            this.configurationLabel.Text = "Configuration ID:";
            // 
            // configuration
            // 
            this.configuration.AccessibleName = "Configuration ID";
            this.configuration.BorderStyle = System.Windows.Forms.BorderStyle.FixedSingle;
            this.configuration.Location = new System.Drawing.Point(186, 60);
            this.configuration.Margin = new System.Windows.Forms.Padding(4);
            this.configuration.Name = "configuration";
            this.configuration.Size = new System.Drawing.Size(200, 31);
            this.configuration.TabIndex = 1;
            // 
            // generalGroupBox
            // 
            this.generalGroupBox.Controls.Add(this.updateChannelLabel);
            this.generalGroupBox.Controls.Add(this.updateChannel);
            this.generalGroupBox.Controls.Add(this.checkUpdate);
            this.generalGroupBox.Controls.Add(this.reportDeviceName);
            this.generalGroupBox.Controls.Add(this.configuration);
            this.generalGroupBox.Controls.Add(this.configurationLabel);
            this.generalGroupBox.Location = new System.Drawing.Point(12, 12);
            this.generalGroupBox.Margin = new System.Windows.Forms.Padding(4);
            this.generalGroupBox.Name = "generalGroupBox";
            this.generalGroupBox.Padding = new System.Windows.Forms.Padding(4);
            this.generalGroupBox.Size = new System.Drawing.Size(712, 335);
            this.generalGroupBox.TabIndex = 3;
            this.generalGroupBox.TabStop = false;
            this.generalGroupBox.Text = "Configuration";
            // 
            // updateChannelLabel
            // 
            this.updateChannelLabel.AutoSize = true;
            this.updateChannelLabel.Location = new System.Drawing.Point(7, 256);
            this.updateChannelLabel.Name = "updateChannelLabel";
            this.updateChannelLabel.Size = new System.Drawing.Size(173, 25);
            this.updateChannelLabel.TabIndex = 6;
            this.updateChannelLabel.Text = "Update Channel:";
            // 
            // updateChannel
            // 
            this.updateChannel.FormattingEnabled = true;
            this.updateChannel.Items.AddRange(new object[] {
            "Stable",
            "Beta"});
            this.updateChannel.Location = new System.Drawing.Point(186, 253);
            this.updateChannel.Name = "updateChannel";
            this.updateChannel.Size = new System.Drawing.Size(166, 33);
            this.updateChannel.TabIndex = 5;
            // 
            // checkUpdate
            // 
            this.checkUpdate.AutoSize = true;
            this.checkUpdate.ForeColor = System.Drawing.SystemColors.ControlText;
            this.checkUpdate.Location = new System.Drawing.Point(11, 209);
            this.checkUpdate.Name = "checkUpdate";
            this.checkUpdate.Size = new System.Drawing.Size(222, 29);
            this.checkUpdate.TabIndex = 4;
            this.checkUpdate.Text = "Check for Updates";
            this.checkUpdate.UseVisualStyleBackColor = true;
            this.checkUpdate.CheckedChanged += new System.EventHandler(this.checkUpdate_CheckedChanged);
            // 
            // reportDeviceName
            // 
            this.reportDeviceName.AutoSize = true;
            this.reportDeviceName.Location = new System.Drawing.Point(11, 109);
            this.reportDeviceName.Margin = new System.Windows.Forms.Padding(4);
            this.reportDeviceName.Name = "reportDeviceName";
            this.reportDeviceName.Size = new System.Drawing.Size(242, 29);
            this.reportDeviceName.TabIndex = 3;
            this.reportDeviceName.Text = "Report Device Name";
            this.reportDeviceName.UseVisualStyleBackColor = true;
            // 
            // save
            // 
            this.save.Location = new System.Drawing.Point(574, 550);
            this.save.Margin = new System.Windows.Forms.Padding(6);
            this.save.Name = "save";
            this.save.Size = new System.Drawing.Size(150, 44);
            this.save.TabIndex = 5;
            this.save.Text = "Save";
            this.save.UseVisualStyleBackColor = true;
            this.save.Click += new System.EventHandler(this.save_Click);
            // 
            // cancel
            // 
            this.cancel.DialogResult = System.Windows.Forms.DialogResult.Cancel;
            this.cancel.Location = new System.Drawing.Point(412, 550);
            this.cancel.Margin = new System.Windows.Forms.Padding(6);
            this.cancel.Name = "cancel";
            this.cancel.Size = new System.Drawing.Size(150, 44);
            this.cancel.TabIndex = 6;
            this.cancel.Text = "Cancel";
            this.cancel.UseVisualStyleBackColor = true;
            this.cancel.Click += new System.EventHandler(this.cancel_Click);
            // 
            // status
            // 
            this.status.AutoSize = true;
            this.status.Location = new System.Drawing.Point(278, 69);
            this.status.Name = "status";
            this.status.Size = new System.Drawing.Size(143, 25);
            this.status.TabIndex = 8;
            this.status.Text = "Disconnected";
            // 
            // statusGroupBox
            // 
            this.statusGroupBox.Controls.Add(this.status);
            this.statusGroupBox.Location = new System.Drawing.Point(13, 355);
            this.statusGroupBox.Name = "statusGroupBox";
            this.statusGroupBox.Size = new System.Drawing.Size(711, 147);
            this.statusGroupBox.TabIndex = 9;
            this.statusGroupBox.TabStop = false;
            this.statusGroupBox.Text = "Status";
            // 
            // SettingsForm
            // 
            this.AcceptButton = this.save;
            this.AutoScaleDimensions = new System.Drawing.SizeF(12F, 25F);
            this.AutoScaleMode = System.Windows.Forms.AutoScaleMode.Font;
            this.CancelButton = this.cancel;
            this.ClientSize = new System.Drawing.Size(736, 609);
            this.Controls.Add(this.statusGroupBox);
            this.Controls.Add(this.cancel);
            this.Controls.Add(this.save);
            this.Controls.Add(this.generalGroupBox);
            this.Margin = new System.Windows.Forms.Padding(4);
            this.MaximizeBox = false;
            this.MinimizeBox = false;
            this.Name = "SettingsForm";
            this.StartPosition = System.Windows.Forms.FormStartPosition.CenterScreen;
            this.Text = "Settings";
            this.systrayContextMenu.ResumeLayout(false);
            this.generalGroupBox.ResumeLayout(false);
            this.generalGroupBox.PerformLayout();
            this.statusGroupBox.ResumeLayout(false);
            this.statusGroupBox.PerformLayout();
            this.ResumeLayout(false);
            this.FormClosing += SettingsForm_FormClosing;
        }

        #endregion

        private System.Windows.Forms.NotifyIcon systray;
        private System.Windows.Forms.ContextMenuStrip systrayContextMenu;
        private System.Windows.Forms.ToolStripMenuItem toggle;
        private System.Windows.Forms.ToolStripMenuItem settings;
        private System.Windows.Forms.ToolStripSeparator toolStripSeparator1;
        private System.Windows.Forms.ToolStripMenuItem quit;
        private System.Windows.Forms.Label configurationLabel;
        private System.Windows.Forms.TextBox configuration;
        private System.Windows.Forms.GroupBox generalGroupBox;
        private System.Windows.Forms.CheckBox reportDeviceName;
        private System.Windows.Forms.Button save;
        private System.Windows.Forms.Button cancel;
        private System.Windows.Forms.CheckBox checkUpdate;
        private System.Windows.Forms.ComboBox updateChannel;
        private System.Windows.Forms.Label updateChannelLabel;
        private System.Windows.Forms.Label status;
        private System.Windows.Forms.GroupBox statusGroupBox;
    }
}

