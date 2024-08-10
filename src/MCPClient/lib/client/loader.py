import importlib
import warnings
from types import ModuleType
from typing import Dict
from typing import Optional


# Keys must use lowercase.
SUPPORTED_MODULES = {
    "assignfileuuids_v0.0": "assign_file_uuids",
    "bindpid_v0.0": "bind_pid",
    "removeunneededfiles_v0.0": "remove_unneeded_files",
    "archivematicaclamscan_v0.0": "archivematica_clamscan",
    "createevent_v0.0": "create_event",
    "examinecontents_v0.0": "examine_contents",
    "fits_v0.0": "fits",
    "identifydspacefiles_v0.0": "identify_dspace_files",
    "identifydspacemetsfiles_v0.0": "identify_dspace_mets_files",
    "identifyfileformat_v0.0": "identify_file_format",
    "ismaildiraip_v0.0": "is_maildir_aip",
    "manualnormalizationcreatemetadataandrestructure_v0.0": "manual_normalization_create_metadata_and_restructure",
    "manualnormalizationidentifyfilesincluded_v0.0": "manual_normalization_identify_files_included",
    "manualnormalizationmoveaccessfilestodip_v0.0": "manual_normalization_move_access_files_to_dip",
    "updatesizeandchecksum_v0.0": "update_size_and_checksum",
    "validatefile_v1.0": "validate_file",
    "createsipfromtransferobjects_v0.0": "create_sip_from_transfer_objects",
    "jsonmetadatatocsv_v0.0": "json_metadata_to_csv",
    "loaddublincore_v0.0": "load_dublin_core",
    "loadlabelsfromcsv_v0.0": "load_labels_from_csv",
    "loadpremiseventsfromxml_v1.0": "load_premis_events_from_xml",
    "manualnormalizationcheckformanualnormalizationdirectory_v0.0": "manual_normalization_check_for_manual_normalization_directory",
    "manualnormalizationremovemndirectories_v0.0": "manual_normalization_remove_mn_directories",
    "movesip_v0.0": "move_sip",
    "moveormerge_v0.0": "move_or_merge",
    "movetransfer_v0.0": "move_transfer",
    "normalize_v1.0": "normalize",
    "normalizereport_v0.0": "normalize_report",
    "parseexternalmets": "parse_external_mets",
    "parsemetstodb_v1.0": "parse_mets_to_db",
    "policycheck_v0.0": "policy_check",
    "removefileswithoutpresmismetadata_v0.0": "remove_files_without_premis_metadata",
    "removehiddenfilesanddirectories_v0.0": "remove_hidden_files_and_directories",
    "restructureforcompliance_v0.0": "restructure_for_compliance",
    "restructureforcompliancesip_v0.0": "restructure_for_compliance_sip",
    "restructureforcompliancemaildir_v0.0": "restructure_for_compliance_maildir",
    "restructurebagaiptosip_v0.0": "restructure_bag_aip_to_sip",
    "retrynormalizeremovenormalized_v0.0": "retry_normalize_remove_normalized",
    "rightsfromcsv_v0.0": "rights_from_csv",
    "changeobjectnames_v0.0": "change_object_names",
    "changesipname_v0.0": "change_sip_name",
    "savedublincore_v0.0": "save_dublin_core",
    "setmaildirfilegrpuseandfileids_v0.0": "set_maildir_file_grp_use_and_file_ids",
    "storefilemodificationdates_v0.0": "store_file_modification_dates",
    "transcribefile_v0.0": "transcribe_file",
    "trimcreaterightsentries_v0.0": "trim_create_rights_entries",
    "trimrestructureforcompliance_v0.0": "trim_restructure_for_compliance",
    "trimverifychecksums_v0.0": "trim_verify_checksums",
    "trimverifymanifest_v0.0": "trim_verify_manifest",
    "verifychecksumsinfilesecofdspacemetsfiles_v0.0": "verify_checksums_in_file_sec_of_dspace_mets_files",
    "verifyaip_v1.0": "verify_aip",
    "verifyandrestructuretransferbag_v0.0": "verify_and_restructure_transfer_bag",
    "verifysipcompliance_v0.0": "verify_sip_compliance",
    "verifytransfercompliance_v0.0": "verify_transfer_compliance",
    "createtransfermets_v1.0": "create_transfer_mets",
    "createmets_v2.0": "create_mets_v2",
    "characterizefile_v0.0": "characterize_file",
    "copytransfersubmissiondocumentation_v0.0": "copy_transfer_submission_documentation",
    "copytransfersmetadataandlogs_v0.0": "copy_transfers_metadata_and_logs",
    "compressaip_v0.0": "compress_aip",
    "archivematicasettransfertype_v0.0": "set_transfer_type",
    "checktransferdirectoryforobjects_v0.0": "check_transfer_directory_for_objects",
    "checkforsubmissiondocumenation_v0.0": "check_for_submission_documentation",
    "checkforservicedirectory_v0.0": "check_for_service_directory",
    "checkforaccessdirectory_v0.0": "check_for_access_directory",
    "declarepids_v0.0": "pid_declaration",
    "bindpids_v0.0": "bind_pids",
    "extractzippedtransfer_v0.0": "extract_zipped_transfer",
    "assignuuidstodirectories_v0.0": "assign_uuids_to_directories",
    "bagit_v0.0": "bag_with_empty_directories",
    "haspackages_v0.0": "has_packages",
    "failedtransfercleanup": "failed_transfer_cleanup",
    "filetofolder_v1.0": "file_to_folder",
    "createtransfermetadata_v0.0": "create_transfer_metadata",
    "extractcontents_v0.0": "extract_contents",
    "extractmaildirattachments_v0.0": "extract_maildir_attachments",
    "failedsipcleanup_v1.0": "failed_sip_cleanup",
    "createsipsfromtrimtransfercontainers_v0.0": "create_sips_from_trim_transfer_containers",
    "determineaipversionkeyexitcode_v0.0": "determine_aip_version_key_exit_code",
    "dipgenerationhelper": "dip_generation_helper",
    "emailfailreport_v0.0": "email_fail_report",
    "verifychecksum_v0.0": "verify_checksum",
    "archivematicaverifymets_v0.0": "verify_mets",
    "copyrecursive_v0.0": "copy_recursive",
    "copysubmissiondocs_v0.0": "copy_submission_docs",
    "copy_v0.0": "cmd_cp",
    "createdirectory_v0.0": "cmd_mkdir",
    "createdirectorytree_v0.0": "cmd_tree",
    "move_v0.0": "cmd_mv",
    "setfilepermission_v0.0": "cmd_chmod",
    "test_v0.0": "cmd_test",
    "storeaip_v0.0": "store_aip",
    "restructuredipforcontentdmupload_v0.0": "restructure_dip_for_content_dm_upload",
    "upload-qubit_v0.0": "upload_qubit",
    "upload-archivesspace_v0.0": "upload_archivesspace",
    "copythumbnailstodipdirectory_v0.0": "copy_thumbnails_to_dip_directory",
    "removedirectories_v0.0": "remove_directories",
    "convertdataversestructure_v0.0": "convert_dataverse_structure",
    "parsedataverse_v0.0": "parse_dataverse_mets",
}


def load_module(module_name: str) -> Optional[ModuleType]:
    # No need to cache here as imports are already cached.
    try:
        return importlib.import_module(f"clientScripts.{module_name}")
    except ImportError as err:
        warnings.warn(
            f"Failed to load client script {module_name}: {err}",
            RuntimeWarning,
            stacklevel=2,
        )
        return None


def get_module_concurrency(module: ModuleType) -> int:
    try:
        return int(module.concurrent_instances())
    except (AttributeError, TypeError, ValueError):
        return 1


def load_job_modules() -> Dict[str, Optional[ModuleType]]:
    """Return a dict of {client script name: module}."""
    supported_modules = SUPPORTED_MODULES

    return dict(
        zip(supported_modules.keys(), map(load_module, supported_modules.values()))
    )
