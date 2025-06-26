import os
import re
import csv


def extract_k6_results(root_dir):
    """
    Extracts http_req_duration and http_req_waiting p(95) values from k6 output files
    within subdirectories of the given root directory.  It also splits the filename
    by underscores and adds the resulting parts as columns.

    Args:
        root_dir (str): The root directory to search for k6 output files.

    Returns:
        list: A list of dictionaries, where each dictionary represents a row in the CSV.
              Each dictionary contains 'dirname', 'filename', 'http_req_duration p95',
              'http_req_waiting p95', and columns for each part of the split filename.
    """

    results = []
    for dirname, _, files in os.walk(root_dir):
        for filename in files:
            if filename.endswith(".txt"):  # or some other k6 output file extension
                filepath = os.path.join(dirname, filename)
                try:
                    with open(filepath, 'r') as f:
                        content = f.read()

                    # Extract http_req_duration p(95)
                    duration_match = re.search(r"http_req_duration.*?p\(95\)=([\d.]+)(ms|s)", content)
                    duration_p95 = float(duration_match.group(1)) if duration_match else None
                    duration_unit = duration_match.group(2) if duration_match else "ms"

                    # Extract http_req_waiting p(95)
                    waiting_match = re.search(r"http_req_waiting.*?p\(95\)=([\d.]+)(ms|s)", content)
                    waiting_p95 = float(waiting_match.group(1)) if waiting_match else None
                    waiting_unit = waiting_match.group(2) if waiting_match else "ms"

                    # Convert to milliseconds if necessary
                    if duration_p95 is not None and duration_unit == "ms":
                        duration_p95 /= 1000.0
                    if waiting_p95 is not None and waiting_unit == "ms":
                        waiting_p95 /= 1000.0

                    # Split filename and create columns
                    filename_without_ext = os.path.splitext(filename)[0]
                    filename_parts = filename_without_ext.split("_")[:-1]

                    results.append({
                        'dirname': os.path.basename(dirname),
                        'operation': filename_parts[0],
                        'count': filename_parts[1],
                        'http_req_duration p(95)': duration_p95,
                        'http_req_waiting p(95)': waiting_p95,
                    })

                except Exception as e:
                    print(f"Error processing {filepath}: {e}")

    return results


def write_csv(results, output_file="k6_results.csv"):
    """
    Writes the extracted k6 results to a CSV file.

    Args:
        results (list): A list of dictionaries containing the k6 results.
        output_file (str): The name of the CSV file to create.
    """

    if not results:
        print("No data to write to CSV.")
        return

    sorted_results = sorted(results, key=lambda x: (x['operation'], x['dirname'], x['count']))

    fieldnames = ['operation', 'dirname', 'count', 'http_req_duration p(95)', 'http_req_waiting p(95)']

    with open(output_file, 'w', newline='') as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(sorted_results)

    print(f"Successfully wrote results to {output_file}")


if __name__ == "__main__":
    root_directory = os.getcwd()  # Use the current directory as the root
    extracted_data = extract_k6_results(root_directory)
    write_csv(extracted_data)
