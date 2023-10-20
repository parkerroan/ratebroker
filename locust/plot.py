import pandas as pd
import numpy as np
import matplotlib.pyplot as plt
import sys

def analyze_and_plot(file_path):
    # Load and process the data
    data = pd.read_csv(file_path)
    
    data['Timestamp'] = pd.to_datetime(data['Timestamp'], format='mixed')

    # Grouping by 'User ID' and 10-second intervals for 'Timestamp'
    user_requests = data.groupby(['User ID', pd.Grouper(key='Timestamp', freq='10S'), 'Status Code']).size().reset_index(name='Count')

    # Calculating success and failure counts
    successes = user_requests[user_requests['Status Code'] == 200]
    failures = user_requests[user_requests['Status Code'] == 429]

    # Calculate the total success and failure counts per interval
    total_successes = successes.groupby('Timestamp')['Count'].sum().reset_index()
    total_failures = failures.groupby('Timestamp')['Count'].sum().reset_index()

    # Adjusting counts to represent rates per second
    successes['Success/Sec'] = successes['Count'] / 10
    failures['Failure/Sec'] = failures['Count'] / 10
    total_successes['Total Success/Sec'] = total_successes['Count'] / 10
    total_failures['Total Failure/Sec'] = total_failures['Count'] / 10

    # Plotting
    plt.figure(figsize=(15, 7))
    user_ids = np.union1d(successes['User ID'].unique(), failures['User ID'].unique())
    colors = plt.cm.viridis(np.linspace(0, 1, len(user_ids) + 2))  # +2 for the total counts

    # Plotting total success and failure rates
    plt.plot(total_successes['Timestamp'], total_successes['Total Success/Sec'], label="Total Success/Sec", color=colors[0], linestyle='-', marker='.')
    plt.plot(total_failures['Timestamp'], total_failures['Total Failure/Sec'], label="Total Failure/Sec", color=colors[1], linestyle='-', marker='.')

    # Plotting per-user success and failure rates
    for i, user_id in enumerate(user_ids, start=2):  # start=2 to skip the first two colors used for the total counts
        user_success = successes[successes['User ID'] == user_id]
        if not user_success.empty:
            plt.plot(user_success['Timestamp'], user_success['Success/Sec'], label=f"Success/Sec: {user_id}", color=colors[i], linestyle='-', marker='o')
        
        user_failure = failures[failures['User ID'] == user_id]
        if not user_failure.empty:
            plt.plot(user_failure['Timestamp'], user_failure['Failure/Sec'], label=f"Failure/Sec: {user_id}", color=colors[i], linestyle='--', marker='x')

    plt.xlabel('Timestamp')
    plt.ylabel('Rate per Second')
    plt.title('Total and Per-User Success and Failure Rate Per Second Over Time')
    plt.legend(loc='upper left', bbox_to_anchor=(1.05, 1))
    plt.grid(True)
    plt.gcf().autofmt_xdate(rotation=45)
    plt.tight_layout()
    plt.show()

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Usage: python script_name.py /path/to/your/log.csv")
    else:
        file_path = sys.argv[1]
        analyze_and_plot(file_path)
